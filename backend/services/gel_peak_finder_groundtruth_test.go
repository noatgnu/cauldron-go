package services

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

type gelGroundTruthProfile struct {
	Source                 string    `json:"source"`
	Profile                []float64 `json:"profile"`
	GroundTruthBandCenters []int     `json:"groundTruthBandCenters"`
	GroundTruthSource      string    `json:"groundTruthSource"`
}

func loadGelGroundTruthProfile(t *testing.T, filename string) gelGroundTruthProfile {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to read ground truth profile %s: %v", filename, err)
	}
	var gt gelGroundTruthProfile
	if err := json.Unmarshal(data, &gt); err != nil {
		t.Fatalf("failed to parse ground truth profile %s: %v", filename, err)
	}
	return gt
}

// matchBandsToGroundTruth greedily pairs each detected position with its nearest unmatched ground-truth position within tolerance.
func matchBandsToGroundTruth(detected []float64, groundTruth []int, tolerance float64) int {
	used := make([]bool, len(groundTruth))
	tp := 0
	for _, d := range detected {
		best := -1
		bestDist := tolerance + 1
		for i, g := range groundTruth {
			if used[i] {
				continue
			}
			dist := math.Abs(d - float64(g))
			if dist <= tolerance && dist < bestDist {
				best = i
				bestDist = dist
			}
		}
		if best >= 0 {
			used[best] = true
			tp++
		}
	}
	return tp
}

// TestFindPeaks_RealGelGroundTruth checks band detection against 25 real ladder lanes across 5 GelGenie/Zenodo scans (DOI 10.5281/zenodo.14641949).
func TestFindPeaks_RealGelGroundTruth(t *testing.T) {
	cases := []struct {
		name         string
		file         string
		minRecall    float64
		minPrecision float64
	}{
		{"14neb_lane1", "gelgenie_14neb_lane1_profile.json", 0.85, 0.95},
		{"14neb_lane2", "gelgenie_14neb_lane2_profile.json", 0.80, 0.90},
		{"13neb_lane1", "gelgenie_13_neb_lane1_profile.json", 0.70, 0.95},
		{"13neb_lane2", "gelgenie_13_neb_lane2_profile.json", 0.70, 0.95},
		{"13neb_lane3", "gelgenie_13_neb_lane3_profile.json", 0.70, 0.95},
		{"13neb_lane4", "gelgenie_13_neb_lane4_profile.json", 0.65, 0.95},
		{"18neb_lane1", "gelgenie_18_neb_lane1_profile.json", 0.80, 0.90},
		{"18neb_lane2", "gelgenie_18_neb_lane2_profile.json", 0.85, 0.95},
		{"18neb_lane3", "gelgenie_18_neb_lane3_profile.json", 0.85, 0.80},
		{"18neb_lane4", "gelgenie_18_neb_lane4_profile.json", 0.85, 0.95},
		{"30neb_lane1", "gelgenie_30_neb_lane1_profile.json", 0.85, 0.85},
		{"30neb_lane2", "gelgenie_30_neb_lane2_profile.json", 0.90, 0.85},
		{"30neb_lane3", "gelgenie_30_neb_lane3_profile.json", 0.85, 0.95},
		{"30neb_lane4", "gelgenie_30_neb_lane4_profile.json", 0.85, 0.95},
		{"30neb_lane5", "gelgenie_30_neb_lane5_profile.json", 0.85, 0.95},
		{"30neb_lane6", "gelgenie_30_neb_lane6_profile.json", 0.90, 0.80},
		{"30neb_lane7", "gelgenie_30_neb_lane7_profile.json", 0.90, 0.95},
		{"30neb_lane8", "gelgenie_30_neb_lane8_profile.json", 0.90, 0.75},
		{"30neb_lane9", "gelgenie_30_neb_lane9_profile.json", 0.90, 0.95},
		{"30neb_lane10", "gelgenie_30_neb_lane10_profile.json", 0.90, 0.85},
		{"30neb_lane11", "gelgenie_30_neb_lane11_profile.json", 0.90, 0.85},
		{"30neb_lane12", "gelgenie_30_neb_lane12_profile.json", 0.75, 0.95},
		{"9thermo_lane1", "gelgenie_9_thermo_lane1_profile.json", 0.85, 0.80},
		{"9thermo_lane2", "gelgenie_9_thermo_lane2_profile.json", 0.65, 0.80},
		{"9thermo_lane3", "gelgenie_9_thermo_lane3_profile.json", 0.65, 0.80},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gt := loadGelGroundTruthProfile(t, c.file)

			baseline := ComputeBaseline(gt.Profile, "rolling-min", "light-bands")
			bands := FindPeaks(gt.Profile, baseline, models.GelPeakParams{Polarity: "light-bands"})

			detected := make([]float64, len(bands))
			for i, b := range bands {
				detected[i] = b.Position
			}

			tp := matchBandsToGroundTruth(detected, gt.GroundTruthBandCenters, 15)
			recall := float64(tp) / float64(len(gt.GroundTruthBandCenters))
			precision := 0.0
			if len(detected) > 0 {
				precision = float64(tp) / float64(len(detected))
			}

			t.Logf("%s: detected=%d groundTruth=%d TP=%d recall=%.2f precision=%.2f", c.name, len(detected), len(gt.GroundTruthBandCenters), tp, recall, precision)

			if recall < c.minRecall {
				t.Errorf("recall %.2f below minimum %.2f (detected %d of %d real bands)", recall, c.minRecall, tp, len(gt.GroundTruthBandCenters))
			}
			if precision < c.minPrecision {
				t.Errorf("precision %.2f below minimum %.2f (%d of %d detections were real)", precision, c.minPrecision, tp, len(detected))
			}
		})
	}
}

// TestFindPeaks_RealGelGroundTruth_DarkBands mirrors TestFindPeaks_RealGelGroundTruth on the same real profiles inverted (max-value minus each point), simulating a genuine dark-bands scan, to confirm dark-bands polarity is exercised against real band shapes too, not just light-bands.
func TestFindPeaks_RealGelGroundTruth_DarkBands(t *testing.T) {
	cases := []struct {
		name         string
		file         string
		minRecall    float64
		minPrecision float64
	}{
		{"lane1", "gelgenie_14neb_lane1_profile.json", 0.85, 0.95},
		{"lane2", "gelgenie_14neb_lane2_profile.json", 0.80, 0.90},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gt := loadGelGroundTruthProfile(t, c.file)

			maxVal := gt.Profile[0]
			for _, v := range gt.Profile {
				if v > maxVal {
					maxVal = v
				}
			}
			inverted := make([]float64, len(gt.Profile))
			for i, v := range gt.Profile {
				inverted[i] = maxVal - v
			}

			baseline := ComputeBaseline(inverted, "rolling-min", "dark-bands")
			bands := FindPeaks(inverted, baseline, models.GelPeakParams{Polarity: "dark-bands"})

			detected := make([]float64, len(bands))
			for i, b := range bands {
				detected[i] = b.Position
			}

			tp := matchBandsToGroundTruth(detected, gt.GroundTruthBandCenters, 15)
			recall := float64(tp) / float64(len(gt.GroundTruthBandCenters))
			precision := 0.0
			if len(detected) > 0 {
				precision = float64(tp) / float64(len(detected))
			}

			t.Logf("%s: detected=%d groundTruth=%d TP=%d recall=%.2f precision=%.2f", c.name, len(detected), len(gt.GroundTruthBandCenters), tp, recall, precision)

			if recall < c.minRecall {
				t.Errorf("recall %.2f below minimum %.2f (detected %d of %d real bands)", recall, c.minRecall, tp, len(gt.GroundTruthBandCenters))
			}
			if precision < c.minPrecision {
				t.Errorf("precision %.2f below minimum %.2f (%d of %d detections were real)", precision, c.minPrecision, tp, len(detected))
			}
		})
	}
}

// TestFindPeaks_EdgeExclusionFraction checks the opt-in edge-exclusion margin removes a real artifact without losing any ground-truth band.
func TestFindPeaks_EdgeExclusionFraction(t *testing.T) {
	gt := loadGelGroundTruthProfile(t, "gelgenie_30_neb_lane8_profile.json")

	baseline := ComputeBaseline(gt.Profile, "rolling-min", "light-bands")
	bands := FindPeaks(gt.Profile, baseline, models.GelPeakParams{Polarity: "light-bands", EdgeExclusionFraction: 0.05})

	for _, b := range bands {
		if b.RelativePosition < 0.05 || b.RelativePosition > 0.95 {
			t.Errorf("band at relative position %.3f should have been excluded by EdgeExclusionFraction", b.RelativePosition)
		}
	}

	detected := make([]float64, len(bands))
	for i, b := range bands {
		detected[i] = b.Position
	}
	tp := matchBandsToGroundTruth(detected, gt.GroundTruthBandCenters, 15)
	if tp != len(gt.GroundTruthBandCenters) {
		t.Errorf("expected all %d real bands to still be found with EdgeExclusionFraction set, got %d", len(gt.GroundTruthBandCenters), tp)
	}
}
