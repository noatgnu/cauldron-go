package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func newTestGelAnalysisService(t *testing.T) (*GelAnalysisService, *DatabaseService) {
	t.Helper()
	db := createTestDB(t)
	ctx := context.WithValue(context.Background(), "wails-test", true)
	settings := NewSettingsService(ctx, db)
	scriptExecutor := NewScriptExecutor(settings, db)
	progress := NewProgressNotifier(context.Background())
	envService := NewEnvironmentService(ctx, db, settings, progress)
	return NewGelAnalysisService(db, scriptExecutor, progress, envService, "v0.0.0-test"), db
}

// writeSingleColumnGray16 writes a width=1, height=len(values) Gray16 PNG, so that
// SumColumnRange(0,0,1,len) returns exactly `values` back. This is the simplest way to drive a
// known intensity profile through the service's public API for integration-style testing.
func writeSingleColumnGray16(t *testing.T, path string, values []uint16) {
	t.Helper()
	img := image.NewGray16(image.Rect(0, 0, 1, len(values)))
	for y, v := range values {
		img.SetGray16(0, y, color.Gray16{Y: v})
	}
	writeTestPNG(t, path, img)
}

func gaussianProfile(length int, base float64, bumps map[int]float64, width float64) []uint16 {
	values := make([]float64, length)
	for i := range values {
		values[i] = base
	}
	for center, height := range bumps {
		for i := range values {
			d := float64(i - center)
			values[i] += height * math.Exp(-(d*d)/(2*width*width))
		}
	}
	out := make([]uint16, length)
	for i, v := range values {
		if v < 0 {
			v = 0
		}
		if v > 65535 {
			v = 65535
		}
		out[i] = uint16(v)
	}
	return out
}

func TestGelAnalysisService_SetLaneAndComputeProfile(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	path := filepath.Join(t.TempDir(), "lane.png")
	values := gaussianProfile(100, 1000, map[int]float64{25: 5000, 75: 4000}, 3)
	writeSingleColumnGray16(t, path, values)

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	lane := models.GelLaneROI{ID: "lane1", Label: "Lane 1", X: 0, Y: 0, Width: 1, Height: 100}
	if err := svc.SetLane(meta.SessionID, lane); err != nil {
		t.Fatalf("SetLane: %v", err)
	}

	lanes, err := svc.GetLanes(meta.SessionID)
	if err != nil || len(lanes) != 1 {
		t.Fatalf("GetLanes: %v, %+v", err, lanes)
	}

	profile, err := svc.ComputeLaneProfile(meta.SessionID, "lane1", models.GelPeakParams{Polarity: "light-bands"})
	if err != nil {
		t.Fatalf("ComputeLaneProfile: %v", err)
	}
	if len(profile.Bands) != 2 {
		t.Fatalf("expected 2 bands, got %d: %+v", len(profile.Bands), profile.Bands)
	}

	if err := svc.RemoveLane(meta.SessionID, "lane1"); err != nil {
		t.Fatalf("RemoveLane: %v", err)
	}
	lanes, _ = svc.GetLanes(meta.SessionID)
	if len(lanes) != 0 {
		t.Errorf("expected 0 lanes after removal, got %d", len(lanes))
	}
}

// write2DGray16 writes a width x height Gray16 PNG from a row-major values slice, for tests that
// need real horizontal structure (e.g. CenterLane), unlike writeSingleColumnGray16's width=1 shape.
func write2DGray16(t *testing.T, path string, width, height int, values []uint16) {
	t.Helper()
	img := image.NewGray16(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetGray16(x, y, color.Gray16{Y: values[y*width+x]})
		}
	}
	writeTestPNG(t, path, img)
}

func TestGelAnalysisService_CenterLane(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	const width, height = 60, 4
	values := make([]uint16, width*height)
	for i := range values {
		values[i] = 60000
	}
	// True lane is a dark band at columns [25,35), centroid 29.5.
	for y := 0; y < height; y++ {
		for x := 25; x < 35; x++ {
			values[y*width+x] = 1000
		}
	}

	path := filepath.Join(t.TempDir(), "offset-lane.png")
	write2DGray16(t, path, width, height, values)

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	// Rough rectangle drawn off to the left, only partially overlapping the true band.
	lane := models.GelLaneROI{ID: "lane1", Label: "Lane 1", X: 20, Y: 0, Width: 10, Height: 4}
	if err := svc.SetLane(meta.SessionID, lane); err != nil {
		t.Fatalf("SetLane: %v", err)
	}

	updated, err := svc.CenterLane(meta.SessionID, "lane1")
	if err != nil {
		t.Fatalf("CenterLane: %v", err)
	}

	// Distance-weighting damps full snap-to-band correction, so expect partial bounded movement, not an exact snap.
	trueX := 29.5 - lane.Width/2
	if updated.X <= lane.X || updated.X >= trueX {
		t.Errorf("centered X = %v, want strictly between original %v and true center %v", updated.X, lane.X, trueX)
	}
	if updated.Width != lane.Width || updated.Height != lane.Height {
		t.Errorf("CenterLane should not change width/height, got %vx%v", updated.Width, updated.Height)
	}
}

func TestGelAnalysisService_DetectBoundary(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	const width, height = 100, 50
	values := make([]uint16, width*height)
	path := filepath.Join(t.TempDir(), "boundary.png")
	write2DGray16(t, path, width, height, values)

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	if err := svc.SetLane(meta.SessionID, models.GelLaneROI{ID: "l1", X: 10, Y: 5, Width: 8, Height: 30}); err != nil {
		t.Fatalf("SetLane: %v", err)
	}
	if err := svc.SetLane(meta.SessionID, models.GelLaneROI{ID: "l2", X: 40, Y: 8, Width: 8, Height: 20}); err != nil {
		t.Fatalf("SetLane: %v", err)
	}

	boundary, err := svc.DetectBoundary(meta.SessionID, 5)
	if err != nil {
		t.Fatalf("DetectBoundary: %v", err)
	}

	if boundary.X != 5 || boundary.Y != 0 || boundary.Width != 48 || boundary.Height != 40 {
		t.Errorf("boundary = %+v, want {X:5 Y:0 Width:48 Height:40}", boundary)
	}

	got, err := svc.GetBoundary(meta.SessionID)
	if err != nil || got == nil || *got != *boundary {
		t.Errorf("GetBoundary = %+v, %v; want %+v, nil", got, err, boundary)
	}

	if err := svc.ClearBoundary(meta.SessionID); err != nil {
		t.Fatalf("ClearBoundary: %v", err)
	}
	got, err = svc.GetBoundary(meta.SessionID)
	if err != nil || got != nil {
		t.Errorf("GetBoundary after clear = %+v, %v; want nil, nil", got, err)
	}
}

func TestGelAnalysisService_DetectBoundary_ClampsToImage(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	const width, height = 100, 50
	values := make([]uint16, width*height)
	path := filepath.Join(t.TempDir(), "boundary-clamp.png")
	write2DGray16(t, path, width, height, values)

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	if err := svc.SetLane(meta.SessionID, models.GelLaneROI{ID: "l1", X: 2, Y: 2, Width: 5, Height: 5}); err != nil {
		t.Fatalf("SetLane: %v", err)
	}

	boundary, err := svc.DetectBoundary(meta.SessionID, 200)
	if err != nil {
		t.Fatalf("DetectBoundary: %v", err)
	}
	if boundary.X != 0 || boundary.Y != 0 || boundary.Width != float64(width) || boundary.Height != float64(height) {
		t.Errorf("boundary = %+v, want clamped to full image {0,0,%d,%d}", boundary, width, height)
	}
}

func TestGelAnalysisService_DetectBoundary_ErrorsWithNoLanes(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	path := filepath.Join(t.TempDir(), "no-lanes.png")
	write2DGray16(t, path, 10, 10, make([]uint16, 100))

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	if _, err := svc.DetectBoundary(meta.SessionID, 5); err == nil {
		t.Error("expected an error when detecting a boundary with no lanes set")
	}
}

func TestGelAnalysisService_SetBoundary(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	path := filepath.Join(t.TempDir(), "manual-boundary.png")
	write2DGray16(t, path, 20, 20, make([]uint16, 400))

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	manual := models.GelBoundary{X: 1, Y: 2, Width: 3, Height: 4}
	if err := svc.SetBoundary(meta.SessionID, manual); err != nil {
		t.Fatalf("SetBoundary: %v", err)
	}

	got, err := svc.GetBoundary(meta.SessionID)
	if err != nil || got == nil || *got != manual {
		t.Errorf("GetBoundary = %+v, %v; want %+v, nil", got, err, manual)
	}
}

func TestGelAnalysisService_FitCalibrationAppliesMW(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	path := filepath.Join(t.TempDir(), "marker.png")
	// Three bands, evenly spaced, decreasing MW top-to-bottom as migration convention dictates.
	values := gaussianProfile(120, 1000, map[int]float64{20: 5000, 60: 5000, 100: 5000}, 3)
	writeSingleColumnGray16(t, path, values)

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	// Evenly spaced in log10(MW) (3, 2, 1) to match the evenly-spaced band positions (20, 60, 100)
	// above, so the fit is expected to be near-perfect.
	markerLane := models.GelLaneROI{
		ID: "marker", Label: "Marker", X: 0, Y: 0, Width: 1, Height: 120,
		IsMarker: true, MarkerMWs: []float64{1000, 100, 10},
	}
	sampleLane := models.GelLaneROI{ID: "sample", Label: "Sample", X: 0, Y: 0, Width: 1, Height: 120}

	if err := svc.SetLane(meta.SessionID, markerLane); err != nil {
		t.Fatalf("SetLane(marker): %v", err)
	}
	if err := svc.SetLane(meta.SessionID, sampleLane); err != nil {
		t.Fatalf("SetLane(sample): %v", err)
	}

	params := models.GelPeakParams{Polarity: "light-bands"}
	if _, err := svc.ComputeAllProfiles(meta.SessionID, params); err != nil {
		t.Fatalf("ComputeAllProfiles: %v", err)
	}

	curve, err := svc.FitCalibrationCurve(meta.SessionID, "marker")
	if err != nil {
		t.Fatalf("FitCalibrationCurve: %v", err)
	}
	if curve.RSquared < 0.99 {
		t.Errorf("expected a near-perfect fit for evenly-spaced synthetic markers, got R2=%v", curve.RSquared)
	}

	profiles, err := svc.ApplyCalibration(meta.SessionID)
	if err != nil {
		t.Fatalf("ApplyCalibration: %v", err)
	}
	sampleProfile := profiles["sample"]
	if len(sampleProfile.Bands) == 0 {
		t.Fatal("expected the sample lane to have bands")
	}
	for _, band := range sampleProfile.Bands {
		if band.MolecularWeight == nil {
			t.Errorf("band %+v missing resolved molecular weight after ApplyCalibration", band)
		}
	}
}

func TestGelAnalysisService_FitCalibration_ErrorsOnBandCountMismatch(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	path := filepath.Join(t.TempDir(), "marker.png")
	values := gaussianProfile(100, 1000, map[int]float64{50: 5000}, 3) // only 1 band
	writeSingleColumnGray16(t, path, values)

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	markerLane := models.GelLaneROI{
		ID: "marker", Label: "Marker", X: 0, Y: 0, Width: 1, Height: 100,
		IsMarker: true, MarkerMWs: []float64{250, 100}, // 2 declared, only 1 detected
	}
	if err := svc.SetLane(meta.SessionID, markerLane); err != nil {
		t.Fatalf("SetLane: %v", err)
	}
	if _, err := svc.ComputeLaneProfile(meta.SessionID, "marker", models.GelPeakParams{Polarity: "light-bands"}); err != nil {
		t.Fatalf("ComputeLaneProfile: %v", err)
	}

	_, err = svc.FitCalibrationCurve(meta.SessionID, "marker")
	if err == nil {
		t.Fatal("expected an error on band-count/MW-count mismatch")
	}
}

func intPtr(v int) *int { return &v }

func TestFindAnchorLanes_SortsByIndexAndComputesCenter(t *testing.T) {
	lanes := map[string]models.GelLaneROI{
		"a": {ID: "a", X: 100, Width: 20, LaneIndex: intPtr(2)},
		"b": {ID: "b", X: 10, Width: 10, LaneIndex: intPtr(0)},
		"c": {ID: "c", X: 50, Width: 30}, // no LaneIndex: not an anchor
	}

	anchors := findAnchorLanes(lanes)

	if len(anchors) != 2 {
		t.Fatalf("expected 2 anchors, got %d: %+v", len(anchors), anchors)
	}
	if anchors[0].Index != 0 || anchors[0].Position != 15 || anchors[0].Width != 10 {
		t.Errorf("anchors[0] = %+v, want {Index:0 Position:15 Width:10}", anchors[0])
	}
	if anchors[1].Index != 2 || anchors[1].Position != 110 || anchors[1].Width != 20 {
		t.Errorf("anchors[1] = %+v, want {Index:2 Position:110 Width:20}", anchors[1])
	}
}

func TestFindAnchorLanes_EmptyWhenNoneIndexed(t *testing.T) {
	lanes := map[string]models.GelLaneROI{
		"a": {ID: "a", X: 100, Width: 20},
	}
	if anchors := findAnchorLanes(lanes); len(anchors) != 0 {
		t.Errorf("expected no anchors, got %+v", anchors)
	}
}

func TestGelAnalysisService_SaveAndReloadSession(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	path := filepath.Join(t.TempDir(), "save-reload.png")
	values := gaussianProfile(50, 1000, map[int]float64{25: 3000}, 3)
	writeSingleColumnGray16(t, path, values)

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}
	lane := models.GelLaneROI{ID: "lane1", Label: "Lane 1", X: 0, Y: 0, Width: 1, Height: 50}
	if err := svc.SetLane(meta.SessionID, lane); err != nil {
		t.Fatalf("SetLane: %v", err)
	}
	boundary := models.GelBoundary{X: 0, Y: 0, Width: 1, Height: 50}
	if err := svc.SetBoundary(meta.SessionID, boundary); err != nil {
		t.Fatalf("SetBoundary: %v", err)
	}

	id, err := svc.SaveSession(meta.SessionID, "My Session")
	if err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	sessions, err := svc.ListSavedSessions()
	if err != nil || len(sessions) != 1 {
		t.Fatalf("ListSavedSessions: %v, %+v", err, sessions)
	}

	reopenedMeta, err := svc.LoadSavedSession(id)
	if err != nil {
		t.Fatalf("LoadSavedSession: %v", err)
	}
	if reopenedMeta.SourcePath != path {
		t.Errorf("reopened sourcePath = %q, want %q", reopenedMeta.SourcePath, path)
	}

	reopenedLanes, err := svc.GetLanes(reopenedMeta.SessionID)
	if err != nil || len(reopenedLanes) != 1 || reopenedLanes[0].ID != "lane1" {
		t.Fatalf("expected the saved lane to be restored, got %v, %+v", err, reopenedLanes)
	}

	reopenedBoundary, err := svc.GetBoundary(reopenedMeta.SessionID)
	if err != nil || reopenedBoundary == nil || *reopenedBoundary != boundary {
		t.Errorf("expected the saved boundary to be restored, got %+v, %v", reopenedBoundary, err)
	}

	if err := svc.DeleteSavedSession(id); err != nil {
		t.Fatalf("DeleteSavedSession: %v", err)
	}
	sessions, _ = svc.ListSavedSessions()
	if len(sessions) != 0 {
		t.Errorf("expected 0 saved sessions after delete, got %d", len(sessions))
	}
}

func TestGelAnalysisService_ExportResultsCSV(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	path := filepath.Join(t.TempDir(), "export.png")
	values := gaussianProfile(60, 1000, map[int]float64{30: 4000}, 3)
	writeSingleColumnGray16(t, path, values)

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}
	lane := models.GelLaneROI{ID: "lane1", Label: "MyLane", X: 0, Y: 0, Width: 1, Height: 60}
	if err := svc.SetLane(meta.SessionID, lane); err != nil {
		t.Fatalf("SetLane: %v", err)
	}
	if _, err := svc.ComputeLaneProfile(meta.SessionID, "lane1", models.GelPeakParams{Polarity: "light-bands"}); err != nil {
		t.Fatalf("ComputeLaneProfile: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "results.csv")
	if err := svc.ExportResultsCSV(meta.SessionID, outPath); err != nil {
		t.Fatalf("ExportResultsCSV: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read exported CSV: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "MyLane") {
		t.Errorf("expected exported CSV to contain lane label 'MyLane', got:\n%s", content)
	}
	if !strings.HasPrefix(content, "lane,bandNumber,position") {
		t.Errorf("expected exported CSV to start with the expected header, got:\n%s", content)
	}
}

func TestGelAnalysisService_GetProvenance(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	path := filepath.Join(t.TempDir(), "provenance.png")
	values := gaussianProfile(80, 1000, map[int]float64{40: 4000}, 3)
	writeSingleColumnGray16(t, path, values)

	fileBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture file: %v", err)
	}
	wantHash := sha256.Sum256(fileBytes)
	wantHashHex := hex.EncodeToString(wantHash[:])

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}
	if meta.ImageSHA256 != wantHashHex {
		t.Errorf("GelImageMeta.ImageSHA256 = %q, want %q", meta.ImageSHA256, wantHashHex)
	}

	lane := models.GelLaneROI{ID: "lane1", Label: "Lane 1", X: 0, Y: 0, Width: 1, Height: 80}
	if err := svc.SetLane(meta.SessionID, lane); err != nil {
		t.Fatalf("SetLane: %v", err)
	}
	params := models.GelPeakParams{Polarity: "light-bands", MinProminence: 0.05}
	if _, err := svc.ComputeLaneProfile(meta.SessionID, "lane1", params); err != nil {
		t.Fatalf("ComputeLaneProfile: %v", err)
	}

	provenance, err := svc.GetProvenance(meta.SessionID)
	if err != nil {
		t.Fatalf("GetProvenance: %v", err)
	}

	if provenance.ImageSHA256 != wantHashHex {
		t.Errorf("provenance.ImageSHA256 = %q, want %q", provenance.ImageSHA256, wantHashHex)
	}
	if provenance.AnalysisEngineVersion != AnalysisEngineVersion {
		t.Errorf("provenance.AnalysisEngineVersion = %q, want %q", provenance.AnalysisEngineVersion, AnalysisEngineVersion)
	}
	if provenance.AppVersion != "v0.0.0-test" {
		t.Errorf("provenance.AppVersion = %q, want %q", provenance.AppVersion, "v0.0.0-test")
	}
	if provenance.PeakParams == nil || *provenance.PeakParams != params {
		t.Errorf("provenance.PeakParams = %+v, want %+v", provenance.PeakParams, params)
	}
	if len(provenance.Lanes) != 1 || provenance.Lanes[0].ID != "lane1" {
		t.Errorf("provenance.Lanes = %+v, want one lane with ID lane1", provenance.Lanes)
	}
	if provenance.GeneratedAt == "" {
		t.Error("provenance.GeneratedAt should not be empty")
	}
	if provenance.AutoDetectUsed {
		t.Error("provenance.AutoDetectUsed should be false when Auto-detect was never run")
	}
}

func TestGelAnalysisService_ExportResultsCSV_WritesProvenanceSidecar(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	path := filepath.Join(t.TempDir(), "export-provenance.png")
	values := gaussianProfile(60, 1000, map[int]float64{30: 4000}, 3)
	writeSingleColumnGray16(t, path, values)

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}
	lane := models.GelLaneROI{ID: "lane1", Label: "MyLane", X: 0, Y: 0, Width: 1, Height: 60}
	if err := svc.SetLane(meta.SessionID, lane); err != nil {
		t.Fatalf("SetLane: %v", err)
	}
	if _, err := svc.ComputeLaneProfile(meta.SessionID, "lane1", models.GelPeakParams{Polarity: "light-bands"}); err != nil {
		t.Fatalf("ComputeLaneProfile: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "results.csv")
	if err := svc.ExportResultsCSV(meta.SessionID, outPath); err != nil {
		t.Fatalf("ExportResultsCSV: %v", err)
	}

	sidecarPath := strings.TrimSuffix(outPath, filepath.Ext(outPath)) + ".provenance.json"
	data, err := os.ReadFile(sidecarPath)
	if err != nil {
		t.Fatalf("expected a provenance sidecar file at %s: %v", sidecarPath, err)
	}

	var provenance models.GelAnalysisProvenance
	if err := json.Unmarshal(data, &provenance); err != nil {
		t.Fatalf("provenance sidecar is not valid JSON: %v", err)
	}
	if provenance.ImageSHA256 != meta.ImageSHA256 {
		t.Errorf("sidecar ImageSHA256 = %q, want %q", provenance.ImageSHA256, meta.ImageSHA256)
	}
}

func TestGelAnalysisService_CloseSession(t *testing.T) {
	svc, _ := newTestGelAnalysisService(t)

	path := filepath.Join(t.TempDir(), "close.png")
	writeSingleColumnGray16(t, path, gaussianProfile(10, 100, nil, 1))

	meta, err := svc.LoadImage(path)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}
	if err := svc.CloseSession(meta.SessionID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}
	if _, err := svc.GetLanes(meta.SessionID); err == nil {
		t.Error("expected an error querying a closed session")
	}
}
