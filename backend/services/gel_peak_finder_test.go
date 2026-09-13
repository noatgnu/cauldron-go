package services

import (
	"math"
	"testing"

	"github.com/noatgnu/cauldron-go/backend/models"
)

func flatProfile(length int, base float64) []float64 {
	values := make([]float64, length)
	for i := range values {
		values[i] = base
	}
	return values
}

func addGaussianBump(values []float64, center int, height float64, width float64) {
	for i := range values {
		d := float64(i - center)
		values[i] += height * math.Exp(-(d*d)/(2*width*width))
	}
}

func TestFindPeaks_SimpleTwoBandProfile(t *testing.T) {
	values := flatProfile(100, 10)
	addGaussianBump(values, 25, 200, 3)
	addGaussianBump(values, 75, 150, 3)

	baseline := ComputeBaseline(values, "rolling-min", "light-bands")
	bands := FindPeaks(values, baseline, models.GelPeakParams{Polarity: "light-bands", MinDistance: 5})

	if len(bands) != 2 {
		t.Fatalf("expected 2 bands, got %d: %+v", len(bands), bands)
	}
	if math.Abs(bands[0].Position-25) > 2 {
		t.Errorf("first band position = %v, want ~25", bands[0].Position)
	}
	if math.Abs(bands[1].Position-75) > 2 {
		t.Errorf("second band position = %v, want ~75", bands[1].Position)
	}
}

func TestFindPeaks_RespectsMinProminence(t *testing.T) {
	values := flatProfile(100, 10)
	addGaussianBump(values, 25, 200, 3)
	addGaussianBump(values, 75, 5, 3) // tiny bump, should be filtered out

	baseline := ComputeBaseline(values, "rolling-min", "light-bands")
	bands := FindPeaks(values, baseline, models.GelPeakParams{Polarity: "light-bands", MinProminence: 0.2, MinDistance: 5})

	if len(bands) != 1 {
		t.Fatalf("expected 1 band after prominence filtering, got %d: %+v", len(bands), bands)
	}
}

func TestFindPeaks_RespectsMinDistance(t *testing.T) {
	values := flatProfile(100, 10)
	addGaussianBump(values, 40, 200, 2)
	addGaussianBump(values, 45, 190, 2) // close to the first, should be suppressed

	baseline := ComputeBaseline(values, "rolling-min", "light-bands")
	bands := FindPeaks(values, baseline, models.GelPeakParams{Polarity: "light-bands", MinDistance: 20})

	if len(bands) != 1 {
		t.Fatalf("expected 1 band after min-distance suppression, got %d: %+v", len(bands), bands)
	}
}

func TestFindPeaks_PolarityInversion(t *testing.T) {
	// dark-bands: bands are dips in a bright background (typical stained gel scan).
	values := flatProfile(100, 200)
	addGaussianBump(values, 50, -180, 3)

	baseline := ComputeBaseline(values, "none", "dark-bands")
	bands := FindPeaks(values, baseline, models.GelPeakParams{Polarity: "dark-bands"})

	if len(bands) != 1 {
		t.Fatalf("expected 1 band, got %d: %+v", len(bands), bands)
	}
	if math.Abs(bands[0].Position-50) > 2 {
		t.Errorf("band position = %v, want ~50", bands[0].Position)
	}
}

func TestFindPeaks_EmptyProfile(t *testing.T) {
	bands := FindPeaks(nil, nil, models.GelPeakParams{})
	if bands != nil {
		t.Errorf("expected nil bands for empty profile, got %+v", bands)
	}
}

func TestFitCalibrationCurve_RecoversKnownSlope(t *testing.T) {
	const wantSlope = -2.5
	const wantIntercept = 3.0

	points := make([]models.GelCalibrationPoint, 0, 6)
	for i := 0; i < 6; i++ {
		pos := float64(i) / 5.0
		logMW := wantSlope*pos + wantIntercept
		points = append(points, models.GelCalibrationPoint{
			Position: pos,
			LogMW:    logMW,
			MW:       math.Pow(10, logMW),
		})
	}

	curve, err := FitCalibrationCurve(points)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(curve.Slope-wantSlope) > 1e-6 {
		t.Errorf("slope = %v, want %v", curve.Slope, wantSlope)
	}
	if math.Abs(curve.Intercept-wantIntercept) > 1e-6 {
		t.Errorf("intercept = %v, want %v", curve.Intercept, wantIntercept)
	}
	if math.Abs(curve.RSquared-1) > 1e-6 {
		t.Errorf("rSquared = %v, want ~1", curve.RSquared)
	}
}

func TestFitCalibrationCurve_ErrorsBelowTwoPoints(t *testing.T) {
	_, err := FitCalibrationCurve([]models.GelCalibrationPoint{{Position: 0, LogMW: 1, MW: 10}})
	if err == nil {
		t.Fatal("expected error for fewer than 2 points")
	}
}

func TestApplyBandOverrides_RemovesExcludedBandAndRecomputesRelativeQuantity(t *testing.T) {
	bands := []models.GelBand{
		{Position: 10, Area: 100, RelativeQuantity: 25},
		{Position: 20, Area: 300, RelativeQuantity: 75},
	}
	overrides := []models.GelBandOverride{
		{ID: "o1", Position: 10, Excluded: true},
	}
	values := flatProfile(30, 10)

	result := ApplyBandOverrides(bands, overrides, values, values, models.GelPeakParams{Polarity: "light-bands"})

	if len(result) != 1 {
		t.Fatalf("expected 1 band, got %d: %+v", len(result), result)
	}
	if result[0].Position != 20 {
		t.Errorf("remaining band position = %v, want 20", result[0].Position)
	}
	if result[0].RelativeQuantity != 100 {
		t.Errorf("remaining band RelativeQuantity = %v, want 100", result[0].RelativeQuantity)
	}
}

func TestApplyBandOverrides_NonExcludedOverrideNearExistingBandIsNotDuplicated(t *testing.T) {
	bands := []models.GelBand{{Position: 10, Area: 100, RelativeQuantity: 100}}
	overrides := []models.GelBandOverride{{ID: "o1", Position: 10, Width: 5, Excluded: false}}
	values := flatProfile(30, 10)

	result := ApplyBandOverrides(bands, overrides, values, values, models.GelPeakParams{Polarity: "light-bands"})

	if len(result) != 1 {
		t.Fatalf("expected no duplicate band near an existing one, got %d: %+v", len(result), result)
	}
}

func TestApplyBandOverrides_NoOverridesReturnsBandsUnchanged(t *testing.T) {
	bands := []models.GelBand{{Position: 10, Area: 100, RelativeQuantity: 100}}

	result := ApplyBandOverrides(bands, nil, nil, nil, models.GelPeakParams{})

	if len(result) != 1 || result[0].Position != 10 {
		t.Fatalf("expected bands unchanged with no overrides, got %+v", result)
	}
}

// TestApplyBandOverrides_ManualAddSynthesizesBandFromProfile covers a real but subtle bump too faint to auto-detect, added manually instead.
func TestApplyBandOverrides_ManualAddSynthesizesBandFromProfile(t *testing.T) {
	values := flatProfile(100, 10)
	addGaussianBump(values, 25, 200, 3)
	addGaussianBump(values, 75, 5, 3) // too faint to auto-detect at the default prominence

	baseline := ComputeBaseline(values, "rolling-min", "light-bands")
	params := models.GelPeakParams{Polarity: "light-bands", MinDistance: 5}
	bands := FindPeaks(values, baseline, params)
	if len(bands) != 1 {
		t.Fatalf("expected only the strong band to auto-detect, got %d: %+v", len(bands), bands)
	}

	overrides := []models.GelBandOverride{{ID: "o1", Position: 75, Width: 10, Excluded: false}}
	result := ApplyBandOverrides(bands, overrides, values, baseline, params)

	if len(result) != 2 {
		t.Fatalf("expected the manual band to be added, got %d: %+v", len(result), result)
	}
	manual := result[1]
	if math.Abs(manual.Position-75) > 0.01 {
		t.Errorf("manual band position = %v, want 75", manual.Position)
	}
	if manual.Intensity <= 0 {
		t.Errorf("manual band intensity = %v, want > 0 (measured from the real bump)", manual.Intensity)
	}
	if manual.Area <= 0 {
		t.Errorf("manual band area = %v, want > 0", manual.Area)
	}
	if manual.RelativeQuantity <= 0 || manual.RelativeQuantity >= result[0].RelativeQuantity {
		t.Errorf("manual band RelativeQuantity = %v, want > 0 and less than the strong band's %v", manual.RelativeQuantity, result[0].RelativeQuantity)
	}
}

func TestApplyCalibrationToProfile(t *testing.T) {
	curve := models.GelCalibrationCurve{Slope: -2, Intercept: 4}
	profile := &models.GelLaneProfile{
		Bands: []models.GelBand{
			{RelativePosition: 0},
			{RelativePosition: 1},
		},
	}

	ApplyCalibrationToProfile(profile, curve)

	if profile.Bands[0].MolecularWeight == nil || math.Abs(*profile.Bands[0].MolecularWeight-math.Pow(10, 4)) > 1e-6 {
		t.Errorf("band 0 MW = %v, want %v", profile.Bands[0].MolecularWeight, math.Pow(10, 4))
	}
	if profile.Bands[1].MolecularWeight == nil || math.Abs(*profile.Bands[1].MolecularWeight-math.Pow(10, 2)) > 1e-6 {
		t.Errorf("band 1 MW = %v, want %v", profile.Bands[1].MolecularWeight, math.Pow(10, 2))
	}
}
