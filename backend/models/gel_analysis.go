package models

import (
	"database/sql/driver"
	"encoding/json"
)

// GelLaneROI is a user-drawn (or auto-detected) lane rectangle in source-image pixel coordinates.
type GelLaneROI struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	X         float64   `json:"x"`
	Y         float64   `json:"y"`
	Width     float64   `json:"width"`
	Height    float64   `json:"height"`
	IsMarker  bool      `json:"isMarker"`
	MarkerMWs []float64 `json:"markerMWs,omitempty"`
}

// GelPeakParams controls smoothing/baseline/peak-detection behavior for ComputeLaneProfile.
type GelPeakParams struct {
	SmoothingWindow int     `json:"smoothingWindow"`
	MinProminence   float64 `json:"minProminence"`
	MinDistance     int     `json:"minDistance"`
	BaselineMethod  string  `json:"baselineMethod"`
	Polarity        string  `json:"polarity"`
}

// GelBand is one detected peak along a lane's intensity profile.
type GelBand struct {
	Position         float64  `json:"position"`
	RelativePosition float64  `json:"relativePosition"`
	Intensity        float64  `json:"intensity"`
	Area             float64  `json:"area"`
	Width            float64  `json:"width"`
	MolecularWeight  *float64 `json:"molecularWeight,omitempty"`
	RelativeQuantity float64  `json:"relativeQuantity"`
}

// GelLaneProfile is one lane's summed intensity profile plus its detected bands.
type GelLaneProfile struct {
	LaneID   string    `json:"laneId"`
	Values   []float64 `json:"values"`
	Baseline []float64 `json:"baseline"`
	Bands    []GelBand `json:"bands"`
}

type GelCalibrationPoint struct {
	Position float64 `json:"position"`
	LogMW    float64 `json:"logMw"`
	MW       float64 `json:"mw"`
}

type GelCalibrationCurve struct {
	Slope     float64               `json:"slope"`
	Intercept float64               `json:"intercept"`
	RSquared  float64               `json:"rSquared"`
	Points    []GelCalibrationPoint `json:"points"`
}

type GelImageMeta struct {
	SessionID   string `json:"sessionId"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	BitDepth    int    `json:"bitDepth"`
	SourcePath  string `json:"sourcePath"`
	ImageSHA256 string `json:"imageSha256"`
}

// GelAnalysisProvenance captures everything needed to audit or attempt to reproduce a result:
// exactly which image bytes were analyzed, with what parameters, by which build of Cauldron, and
// — if Auto-detect was used — which Python interpreter/package versions and script produced it.
type GelAnalysisProvenance struct {
	GeneratedAt            string               `json:"generatedAt"` // RFC3339
	AppVersion             string               `json:"appVersion"`
	AnalysisEngineVersion  string               `json:"analysisEngineVersion"`
	VCSRevision            string               `json:"vcsRevision,omitempty"`
	VCSModified            bool                 `json:"vcsModified,omitempty"`
	ImagePath              string               `json:"imagePath"`
	ImageSHA256            string               `json:"imageSha256"`
	Lanes                  []GelLaneROI         `json:"lanes"`
	PeakParams             *GelPeakParams       `json:"peakParams,omitempty"`
	Calibration            *GelCalibrationCurve `json:"calibration,omitempty"`
	AutoDetectUsed         bool                 `json:"autoDetectUsed"`
	PythonVersion          string               `json:"pythonVersion,omitempty"`
	PythonPackages         []string             `json:"pythonPackages,omitempty"`
	AutoDetectScriptSHA256 string               `json:"autoDetectScriptSha256,omitempty"`
}

// GelLaneROIList is a JSON-in-TEXT-column type, mirroring StringArray in job.go.
type GelLaneROIList []GelLaneROI

func (l *GelLaneROIList) Scan(value interface{}) error {
	if value == nil {
		*l = []GelLaneROI{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		*l = []GelLaneROI{}
		return nil
	}

	return json.Unmarshal(bytes, l)
}

func (l GelLaneROIList) Value() (driver.Value, error) {
	if len(l) == 0 {
		return "[]", nil
	}
	return json.Marshal(l)
}

// GelAnalysisSession is a persisted "recipe" (image path + lane ROIs + peak params) so a gel
// analysis session can be reopened later without storing decoded pixel data. It also carries a
// provenance snapshot taken at save time (image hash, app/engine version, auto-detect Python
// environment) so a reviewer reopening it later can check what produced it.
//
// Schema changes to this struct must stay additive (new optional fields only) — older saved
// sessions must keep loading correctly in newer app versions; never rename/remove a column.
type GelAnalysisSession struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"not null" json:"name"`
	ImagePath      string         `gorm:"not null" json:"imagePath"`
	Lanes          GelLaneROIList `gorm:"type:text" json:"lanes"`
	PeakParams     string         `gorm:"type:text" json:"peakParams"`
	Notes          string         `json:"notes"`
	ImageSHA256    string         `json:"imageSha256"`
	AppVersion     string         `json:"appVersion"`
	EngineVersion  string         `json:"engineVersion"`
	AutoDetectUsed bool           `json:"autoDetectUsed"`
	PythonVersion  string         `json:"pythonVersion"`
	PythonPackages StringArray    `gorm:"type:text" json:"pythonPackages"`
	CreatedAt      int64          `gorm:"not null" json:"createdAt"`
	UpdatedAt      int64          `gorm:"not null" json:"updatedAt"`
}
