package services

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/noatgnu/cauldron-go/backend/models"
)

// gelSessionState is the in-memory-only working state for one loaded gel image. Pixel data and
// computed profiles are intentionally never persisted to SQLite — they're large and purely
// derived from the source file; only the lightweight "recipe" (GelAnalysisSession) is saved.
type gelSessionState struct {
	imagePath   string
	imageHash   string
	image       *GelImageBuffer
	lanes       map[string]models.GelLaneROI
	profiles    map[string]models.GelLaneProfile
	calibration *models.GelCalibrationCurve
	cancel      context.CancelFunc

	lastPeakParams *models.GelPeakParams

	autoDetectUsed         bool
	pythonVersion          string
	pythonPackages         []string
	autoDetectScriptSHA256 string
}

// GelAutoDetectResult is what the bundled Python auto-detect script hands back.
type GelAutoDetectResult struct {
	Lanes        []models.GelLaneROI `json:"lanes"`
	DeskewAngle  float64             `json:"deskewAngle"`
	UsedFallback bool                `json:"usedFallback"`
}

type gelAutoDetectScriptOutput struct {
	DeskewAngle float64 `json:"deskewAngle"`
	Lanes       []struct {
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
		Index  int     `json:"index"`
	} `json:"lanes"`
	BackgroundSubtractedImage string `json:"backgroundSubtractedImage"`
}

// GelAnalysisService implements interactive gel/Western blot densitometry: manual lane ROIs,
// intensity-profile peak-finding, and molecular-weight calibration, all synchronous and
// in-process (not routed through JobQueueService/PluginLoaderV2 — see the feature's plan doc for
// why). An optional Auto-detect action shells out to a bundled Python script via the same
// ScriptExecutor plugins use, under the synthetic type key "gel-analysis".
type GelAnalysisService struct {
	db             *DatabaseService
	scriptExecutor *ScriptExecutor
	progress       *ProgressNotifier
	envService     *EnvironmentService
	appVersion     string

	mu       sync.RWMutex
	sessions map[string]*gelSessionState
}

func NewGelAnalysisService(db *DatabaseService, scriptExecutor *ScriptExecutor, progress *ProgressNotifier, envService *EnvironmentService, appVersion string) *GelAnalysisService {
	return &GelAnalysisService{
		db:             db,
		scriptExecutor: scriptExecutor,
		progress:       progress,
		envService:     envService,
		appVersion:     appVersion,
		sessions:       make(map[string]*gelSessionState),
	}
}

func (s *GelAnalysisService) getSession(sessionID string) (*gelSessionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("gel analysis session %q not found", sessionID)
	}
	return session, nil
}

// LoadImage decodes a gel scan and starts a new in-memory session for it.
func (s *GelAnalysisService) LoadImage(path string) (*models.GelImageMeta, error) {
	buf, meta, err := LoadGelImageFile(path)
	if err != nil {
		return nil, err
	}

	sessionID := uuid.New().String()
	meta.SessionID = sessionID

	s.mu.Lock()
	s.sessions[sessionID] = &gelSessionState{
		imagePath: path,
		imageHash: meta.ImageSHA256,
		image:     buf,
		lanes:     make(map[string]models.GelLaneROI),
		profiles:  make(map[string]models.GelLaneProfile),
	}
	s.mu.Unlock()

	return &meta, nil
}

// CloseSession discards a session's in-memory state.
func (s *GelAnalysisService) CloseSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.sessions[sessionID]; ok && session.cancel != nil {
		session.cancel()
	}
	delete(s.sessions, sessionID)
	return nil
}

// GetImagePreview returns an 8-bit contrast-stretched PNG of the session's working image.
func (s *GelAnalysisService) GetImagePreview(sessionID string) ([]byte, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}
	return EncodePreviewPNG(session.image)
}

// SetLane creates or updates a lane ROI.
func (s *GelAnalysisService) SetLane(sessionID string, lane models.GelLaneROI) error {
	session, err := s.getSession(sessionID)
	if err != nil {
		return err
	}
	if lane.ID == "" {
		return fmt.Errorf("lane ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	session.lanes[lane.ID] = lane
	return nil
}

// RemoveLane deletes a lane ROI and its computed profile, if any.
func (s *GelAnalysisService) RemoveLane(sessionID, laneID string) error {
	session, err := s.getSession(sessionID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(session.lanes, laneID)
	delete(session.profiles, laneID)
	return nil
}

// GetLanes lists a session's current lane ROIs.
func (s *GelAnalysisService) GetLanes(sessionID string) ([]models.GelLaneROI, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	lanes := make([]models.GelLaneROI, 0, len(session.lanes))
	for _, l := range session.lanes {
		lanes = append(lanes, l)
	}
	return lanes, nil
}

// ComputeLaneProfile sums pixel intensity for one lane's ROI and runs peak-finding on it.
func (s *GelAnalysisService) ComputeLaneProfile(sessionID, laneID string, params models.GelPeakParams) (*models.GelLaneProfile, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	lane, ok := session.lanes[laneID]
	if !ok {
		return nil, fmt.Errorf("lane %q not found", laneID)
	}

	values := session.image.SumColumnRange(int(lane.X), int(lane.Y), int(lane.X+lane.Width), int(lane.Y+lane.Height))
	baseline := ComputeBaseline(values, params.BaselineMethod)
	bands := FindPeaks(values, baseline, params)

	profile := models.GelLaneProfile{
		LaneID:   laneID,
		Values:   values,
		Baseline: baseline,
		Bands:    bands,
	}
	session.profiles[laneID] = profile

	paramsCopy := params
	session.lastPeakParams = &paramsCopy

	return &profile, nil
}

// ComputeAllProfiles recomputes every lane's profile with the given peak params.
func (s *GelAnalysisService) ComputeAllProfiles(sessionID string, params models.GelPeakParams) (map[string]models.GelLaneProfile, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	laneIDs := make([]string, 0, len(session.lanes))
	for id := range session.lanes {
		laneIDs = append(laneIDs, id)
	}
	s.mu.RUnlock()

	for _, id := range laneIDs {
		if _, err := s.ComputeLaneProfile(sessionID, id, params); err != nil {
			return nil, err
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]models.GelLaneProfile, len(session.profiles))
	for id, p := range session.profiles {
		out[id] = p
	}
	return out, nil
}

// FitCalibrationCurve pairs the marker lane's detected bands (top-to-bottom) 1:1 with its
// declared MarkerMWs (also top-to-bottom, since larger proteins migrate less) and fits a
// log10(MW)-vs-migration-distance line. It requires the counts to match.
func (s *GelAnalysisService) FitCalibrationCurve(sessionID, markerLaneID string) (*models.GelCalibrationCurve, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	lane, ok := session.lanes[markerLaneID]
	if !ok {
		return nil, fmt.Errorf("lane %q not found", markerLaneID)
	}
	if !lane.IsMarker {
		return nil, fmt.Errorf("lane %q is not marked as a ladder/marker lane", markerLaneID)
	}
	profile, ok := session.profiles[markerLaneID]
	if !ok {
		return nil, fmt.Errorf("lane %q has no computed profile yet — compute its profile first", markerLaneID)
	}
	if len(profile.Bands) != len(lane.MarkerMWs) {
		return nil, fmt.Errorf(
			"detected %d bands in the marker lane but %d known molecular weights were provided — adjust peak detection parameters or marker weights so the counts match",
			len(profile.Bands), len(lane.MarkerMWs),
		)
	}

	points := make([]models.GelCalibrationPoint, len(profile.Bands))
	for i, band := range profile.Bands {
		mw := lane.MarkerMWs[i]
		points[i] = models.GelCalibrationPoint{
			Position: band.RelativePosition,
			LogMW:    math.Log10(mw),
			MW:       mw,
		}
	}

	curve, err := FitCalibrationCurve(points)
	if err != nil {
		return nil, err
	}
	session.calibration = &curve
	return &curve, nil
}

// ApplyCalibration resolves MolecularWeight for every band in every computed profile using the
// session's fitted calibration curve.
func (s *GelAnalysisService) ApplyCalibration(sessionID string) (map[string]models.GelLaneProfile, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if session.calibration == nil {
		return nil, fmt.Errorf("no calibration curve has been fitted for this session yet")
	}

	for id, profile := range session.profiles {
		ApplyCalibrationToProfile(&profile, *session.calibration)
		session.profiles[id] = profile
	}

	out := make(map[string]models.GelLaneProfile, len(session.profiles))
	for id, p := range session.profiles {
		out[id] = p
	}
	return out, nil
}

// GetProvenance assembles a complete audit record for the session's current state: the exact
// image bytes analyzed (by hash), the parameters and lane ROIs that produced the current
// profiles/calibration, this build's version info, and — if Auto-detect was used — the Python
// environment and script that ran. This is what a reviewer checks to verify a result wasn't
// produced by externally manipulating the image, and what they'd need to attempt to reproduce it.
func (s *GelAnalysisService) GetProvenance(sessionID string) (*models.GelAnalysisProvenance, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	lanes := make([]models.GelLaneROI, 0, len(session.lanes))
	for _, l := range session.lanes {
		lanes = append(lanes, l)
	}

	provenance := &models.GelAnalysisProvenance{
		GeneratedAt:            time.Now().UTC().Format(time.RFC3339),
		AppVersion:             s.appVersion,
		AnalysisEngineVersion:  AnalysisEngineVersion,
		ImagePath:              session.imagePath,
		ImageSHA256:            session.imageHash,
		Lanes:                  lanes,
		PeakParams:             session.lastPeakParams,
		Calibration:            session.calibration,
		AutoDetectUsed:         session.autoDetectUsed,
		PythonVersion:          session.pythonVersion,
		PythonPackages:         session.pythonPackages,
		AutoDetectScriptSHA256: session.autoDetectScriptSHA256,
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				provenance.VCSRevision = setting.Value
			case "vcs.modified":
				provenance.VCSModified = setting.Value == "true"
			}
		}
	}

	return provenance, nil
}

// ExportResultsTable flattens every lane's profile/bands into rows suitable for a UI table or CSV.
func (s *GelAnalysisService) ExportResultsTable(sessionID string) ([]map[string]interface{}, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows []map[string]interface{}
	for laneID, profile := range session.profiles {
		lane := session.lanes[laneID]
		for i, band := range profile.Bands {
			row := map[string]interface{}{
				"lane":             lane.Label,
				"bandNumber":       i + 1,
				"position":         band.Position,
				"relativePosition": band.RelativePosition,
				"intensity":        band.Intensity,
				"area":             band.Area,
				"relativeQuantity": band.RelativeQuantity,
			}
			if band.MolecularWeight != nil {
				row["molecularWeight"] = *band.MolecularWeight
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// ExportResultsCSV writes the flattened results table to a CSV file, plus — best-effort — a
// sibling <name>.provenance.json audit manifest (see GetProvenance) so a reviewer always gets the
// metadata needed to verify or attempt to reproduce the exported results.
func (s *GelAnalysisService) ExportResultsCSV(sessionID, outputPath string) error {
	rows, err := s.ExportResultsTable(sessionID)
	if err != nil {
		return err
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"lane", "bandNumber", "position", "relativePosition", "intensity", "area", "relativeQuantity", "molecularWeight"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, row := range rows {
		record := make([]string, len(header))
		for i, col := range header {
			if v, ok := row[col]; ok {
				record[i] = fmt.Sprintf("%v", v)
			}
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	if err := writer.Error(); err != nil {
		return err
	}

	s.writeProvenanceSidecar(sessionID, outputPath)
	return nil
}

// writeProvenanceSidecar writes <outputPath without ext>.provenance.json next to a results
// export. Best-effort: a failure here is logged, not returned, since the export's core job (the
// data) already succeeded.
func (s *GelAnalysisService) writeProvenanceSidecar(sessionID, outputPath string) {
	provenance, err := s.GetProvenance(sessionID)
	if err != nil {
		log.Printf("[GelAnalysisService] Could not assemble provenance for export: %v", err)
		return
	}

	data, err := json.MarshalIndent(provenance, "", "  ")
	if err != nil {
		log.Printf("[GelAnalysisService] Could not marshal provenance for export: %v", err)
		return
	}

	ext := filepath.Ext(outputPath)
	sidecarPath := strings.TrimSuffix(outputPath, ext) + ".provenance.json"
	if err := os.WriteFile(sidecarPath, data, 0644); err != nil {
		log.Printf("[GelAnalysisService] Could not write provenance sidecar: %v", err)
	}
}

// SaveSession persists the current lane ROIs and peak params as a reopenable "recipe" (no pixel
// data — reopening re-decodes the image from ImagePath).
func (s *GelAnalysisService) SaveSession(sessionID, name string) (uint, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return 0, err
	}

	s.mu.RLock()
	lanes := make(models.GelLaneROIList, 0, len(session.lanes))
	for _, l := range session.lanes {
		lanes = append(lanes, l)
	}
	imagePath := session.imagePath
	imageHash := session.imageHash
	autoDetectUsed := session.autoDetectUsed
	pythonVersion := session.pythonVersion
	pythonPackages := session.pythonPackages
	s.mu.RUnlock()

	now := time.Now().Unix()
	record := models.GelAnalysisSession{
		Name:           name,
		ImagePath:      imagePath,
		Lanes:          lanes,
		ImageSHA256:    imageHash,
		AppVersion:     s.appVersion,
		EngineVersion:  AnalysisEngineVersion,
		AutoDetectUsed: autoDetectUsed,
		PythonVersion:  pythonVersion,
		PythonPackages: models.StringArray(pythonPackages),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.db.GetDB().Create(&record).Error; err != nil {
		return 0, fmt.Errorf("failed to save gel analysis session: %w", err)
	}
	return record.ID, nil
}

// LoadSavedSession re-decodes the saved session's image and replays its lane ROIs into a fresh
// in-memory session.
func (s *GelAnalysisService) LoadSavedSession(id uint) (*models.GelImageMeta, error) {
	var record models.GelAnalysisSession
	if err := s.db.GetDB().First(&record, id).Error; err != nil {
		return nil, fmt.Errorf("failed to load gel analysis session: %w", err)
	}

	meta, err := s.LoadImage(record.ImagePath)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	session := s.sessions[meta.SessionID]
	for _, lane := range record.Lanes {
		session.lanes[lane.ID] = lane
	}
	s.mu.Unlock()

	return meta, nil
}

// ListSavedSessions lists all saved gel analysis session recipes, newest first.
func (s *GelAnalysisService) ListSavedSessions() ([]models.GelAnalysisSession, error) {
	var records []models.GelAnalysisSession
	if err := s.db.GetDB().Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to list gel analysis sessions: %w", err)
	}
	return records, nil
}

// DeleteSavedSession removes a saved session recipe.
func (s *GelAnalysisService) DeleteSavedSession(id uint) error {
	return s.db.GetDB().Delete(&models.GelAnalysisSession{}, id).Error
}

// RunAutoDetect writes the session's working image to a temp 16-bit PNG and runs the bundled
// Python auto-detect script against it, synchronously, using the same ScriptExecutor plugins
// use. jobID is a synthetic bookkeeping string only — it's never registered with
// JobQueueService, so the script executor's optional job-update callbacks are safe no-ops for it.
func (s *GelAnalysisService) RunAutoDetect(sessionID, jobID string) (*GelAutoDetectResult, error) {
	session, err := s.getSession(sessionID)
	if err != nil {
		return nil, err
	}

	progressID := "gel-auto-detect:" + sessionID
	s.progress.EmitStart(ProgressTypeAnalysis, progressID, "Starting gel auto-detect...")

	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	session.cancel = cancel
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		session.cancel = nil
		s.mu.Unlock()
	}()

	outputDir, err := os.MkdirTemp("", "gel-auto-detect-*")
	if err != nil {
		s.progress.EmitError(ProgressTypeAnalysis, progressID, "Failed to create working directory", err.Error())
		return nil, fmt.Errorf("failed to create temp output directory: %w", err)
	}
	defer os.RemoveAll(outputDir)

	pngBytes, err := EncodeGray16PNG(session.image)
	if err != nil {
		s.progress.EmitError(ProgressTypeAnalysis, progressID, "Failed to encode working image", err.Error())
		return nil, fmt.Errorf("failed to encode working image: %w", err)
	}
	inputPath := filepath.Join(outputDir, "input.png")
	if err := os.WriteFile(inputPath, pngBytes, 0644); err != nil {
		s.progress.EmitError(ProgressTypeAnalysis, progressID, "Failed to write working image", err.Error())
		return nil, fmt.Errorf("failed to write temp input image: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		s.progress.EmitError(ProgressTypeAnalysis, progressID, "Failed to resolve executable path", err.Error())
		return nil, fmt.Errorf("failed to resolve executable path: %w", err)
	}
	scriptFolder := filepath.Join(filepath.Dir(exePath), "scripts", "gel-analysis")

	s.captureAutoDetectProvenance(session, scriptFolder)

	s.progress.EmitProgress(ProgressTypeAnalysis, progressID, "Running Python auto-detect...", 30)

	err = s.scriptExecutor.ExecutePythonScript(ctx, jobID, ScriptConfig{
		Type:       "gel-analysis",
		FolderPath: scriptFolder,
		ScriptName: "auto_detect.py",
		Args:       []string{"--input", inputPath, "--output", outputDir},
	})
	if err != nil {
		s.progress.EmitError(ProgressTypeAnalysis, progressID, "Auto-detect failed", err.Error())
		return nil, fmt.Errorf("auto-detect script failed: %w", err)
	}

	lanesJSONPath := filepath.Join(outputDir, "lanes.json")
	data, err := os.ReadFile(lanesJSONPath)
	if err != nil {
		s.progress.EmitError(ProgressTypeAnalysis, progressID, "Auto-detect produced no output", err.Error())
		return nil, fmt.Errorf("failed to read auto-detect output: %w", err)
	}

	var scriptOut gelAutoDetectScriptOutput
	if err := json.Unmarshal(data, &scriptOut); err != nil {
		s.progress.EmitError(ProgressTypeAnalysis, progressID, "Failed to parse auto-detect output", err.Error())
		return nil, fmt.Errorf("failed to parse auto-detect output: %w", err)
	}

	result := &GelAutoDetectResult{DeskewAngle: scriptOut.DeskewAngle}
	for _, l := range scriptOut.Lanes {
		result.Lanes = append(result.Lanes, models.GelLaneROI{
			ID:     uuid.New().String(),
			Label:  fmt.Sprintf("Lane %d", l.Index+1),
			X:      l.X,
			Y:      l.Y,
			Width:  l.Width,
			Height: l.Height,
		})
	}

	if scriptOut.BackgroundSubtractedImage != "" {
		bgPath := filepath.Join(outputDir, scriptOut.BackgroundSubtractedImage)
		if buf, _, err := LoadGelImageFile(bgPath); err == nil {
			s.mu.Lock()
			session.image = buf
			s.mu.Unlock()
		} else {
			result.UsedFallback = true
		}
	}

	s.progress.EmitComplete(ProgressTypeAnalysis, progressID, fmt.Sprintf("Auto-detect found %d lanes", len(result.Lanes)))
	return result, nil
}

// captureAutoDetectProvenance records which script and Python environment are about to run, for
// later inclusion in GetProvenance. Best-effort — failures here are logged, not fatal, since the
// analysis result matters more than its metadata.
func (s *GelAnalysisService) captureAutoDetectProvenance(session *gelSessionState, scriptFolder string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session.autoDetectUsed = true

	scriptPath := filepath.Join(scriptFolder, "auto_detect.py")
	if data, err := os.ReadFile(scriptPath); err == nil {
		hash := sha256.Sum256(data)
		session.autoDetectScriptSHA256 = hex.EncodeToString(hash[:])
	} else {
		log.Printf("[GelAnalysisService] Could not hash auto_detect.py for provenance: %v", err)
	}

	binding, err := s.db.GetPluginEnvironmentBinding("gel-analysis", "python")
	if err != nil || binding == nil {
		if err != nil {
			log.Printf("[GelAnalysisService] Could not resolve Python environment binding for provenance: %v", err)
		}
		return
	}

	if out, err := exec.Command(binding.EnvironmentPath, "--version").CombinedOutput(); err == nil {
		session.pythonVersion = strings.TrimSpace(string(out))
	} else {
		log.Printf("[GelAnalysisService] Could not capture Python version for provenance: %v", err)
	}

	if s.envService != nil {
		if packages, err := s.envService.ListPythonPackages(binding.EnvironmentPath); err == nil {
			session.pythonPackages = packages
		} else {
			log.Printf("[GelAnalysisService] Could not list Python packages for provenance: %v", err)
		}
	}
}

// CancelAutoDetect cancels sessionID's in-flight auto-detect run, if any. The frontend only ever
// knows the sessionID (RunAutoDetect's synthetic jobID is purely internal bookkeeping), so
// cancellation is keyed off the session's own stored context.CancelFunc rather than the jobID.
func (s *GelAnalysisService) CancelAutoDetect(sessionID string) error {
	session, err := s.getSession(sessionID)
	if err != nil {
		return err
	}

	s.mu.RLock()
	cancel := session.cancel
	s.mu.RUnlock()

	if cancel == nil {
		return fmt.Errorf("no auto-detect is currently running for this session")
	}
	cancel()
	return nil
}
