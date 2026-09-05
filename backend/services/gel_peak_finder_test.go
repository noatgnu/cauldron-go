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

	baseline := ComputeBaseline(values, "rolling-min")
	bands := FindPeaks(values, baseline, models.GelPeakParams{Polarity: "light-bands"})

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

	baseline := ComputeBaseline(values, "rolling-min")
	bands := FindPeaks(values, baseline, models.GelPeakParams{Polarity: "light-bands", MinProminence: 0.2})

	if len(bands) != 1 {
		t.Fatalf("expected 1 band after prominence filtering, got %d: %+v", len(bands), bands)
	}
}

func TestFindPeaks_RespectsMinDistance(t *testing.T) {
	values := flatProfile(100, 10)
	addGaussianBump(values, 40, 200, 2)
	addGaussianBump(values, 45, 190, 2) // close to the first, should be suppressed

	baseline := ComputeBaseline(values, "rolling-min")
	bands := FindPeaks(values, baseline, models.GelPeakParams{Polarity: "light-bands", MinDistance: 20})

	if len(bands) != 1 {
		t.Fatalf("expected 1 band after min-distance suppression, got %d: %+v", len(bands), bands)
	}
}

func TestFindPeaks_PolarityInversion(t *testing.T) {
	// dark-bands: bands are dips in a bright background (typical stained gel scan).
	values := flatProfile(100, 200)
	addGaussianBump(values, 50, -180, 3)

	baseline := ComputeBaseline(values, "none")
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
