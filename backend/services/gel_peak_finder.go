package services

import (
	"fmt"
	"math"
	"sort"

	"github.com/noatgnu/cauldron-go/backend/models"
)

// AnalysisEngineVersion identifies the behavior of this file's peak-finding/calibration math.
// Bump it only when SmoothProfile, ComputeBaseline, FindPeaks, or FitCalibrationCurve's *behavior*
// changes in a way that would alter results for the same inputs — not for unrelated app changes
// (UI, bugfixes elsewhere). This is what a reviewer checks, alongside the image hash, to know
// whether a result was produced by the same computational method as a prior run.
const AnalysisEngineVersion = "1.0.0"

// SmoothProfile applies a centered moving average of the given odd window size. A window of 0
// or 1 disables smoothing (returns a copy of values unchanged).
func SmoothProfile(values []float64, window int) []float64 {
	out := make([]float64, len(values))
	if window <= 1 || len(values) == 0 {
		copy(out, values)
		return out
	}

	half := window / 2
	for i := range values {
		lo := i - half
		if lo < 0 {
			lo = 0
		}
		hi := i + half
		if hi >= len(values) {
			hi = len(values) - 1
		}
		var sum float64
		for j := lo; j <= hi; j++ {
			sum += values[j]
		}
		out[i] = sum / float64(hi-lo+1)
	}
	return out
}

// ComputeBaseline estimates a local background for a profile, to be subtracted before peak
// detection. "rolling-min" (default) uses a sliding-window minimum; "percentile" uses the 5th
// percentile within the same window, which tolerates a little noise better than a hard minimum;
// "none" returns an all-zero baseline (no correction).
func ComputeBaseline(values []float64, method string) []float64 {
	baseline := make([]float64, len(values))
	if len(values) == 0 {
		return baseline
	}
	if method == "" {
		method = "rolling-min"
	}
	if method == "none" {
		return baseline
	}

	window := len(values) / 8
	if window < 3 {
		window = 3
	}
	half := window / 2

	for i := range values {
		lo := i - half
		if lo < 0 {
			lo = 0
		}
		hi := i + half
		if hi >= len(values) {
			hi = len(values) - 1
		}
		windowVals := append([]float64(nil), values[lo:hi+1]...)
		sort.Float64s(windowVals)

		switch method {
		case "percentile":
			idx := int(float64(len(windowVals)-1) * 0.05)
			baseline[i] = windowVals[idx]
		default: // "rolling-min"
			baseline[i] = windowVals[0]
		}
	}
	return baseline
}

// FindPeaks detects bands in a baseline-corrected, optionally polarity-inverted intensity
// profile. It's a small, dependency-free equivalent of scipy.signal.find_peaks(distance=...):
// local maxima -> prominence filter -> greedy minimum-distance non-max suppression.
func FindPeaks(values, baseline []float64, params models.GelPeakParams) []models.GelBand {
	if len(values) == 0 {
		return nil
	}

	smoothWindow := params.SmoothingWindow
	if smoothWindow == 0 {
		smoothWindow = 7
	}
	smoothed := SmoothProfile(values, smoothWindow)

	if params.Polarity == "light-bands" {
		// leave as-is: bands are already maxima
	} else {
		maxVal := smoothed[0]
		for _, v := range smoothed {
			if v > maxVal {
				maxVal = v
			}
		}
		inverted := make([]float64, len(smoothed))
		for i, v := range smoothed {
			inverted[i] = maxVal - v
		}
		smoothed = inverted
	}

	corrected := make([]float64, len(smoothed))
	profileMax := 0.0
	for i := range smoothed {
		v := smoothed[i]
		if i < len(baseline) {
			v -= baseline[i]
		}
		if v < 0 {
			v = 0
		}
		corrected[i] = v
		if v > profileMax {
			profileMax = v
		}
	}
	if profileMax == 0 {
		return nil
	}

	minProminence := params.MinProminence
	if minProminence <= 0 {
		minProminence = 0.05
	}
	minDistance := params.MinDistance
	if minDistance <= 0 {
		minDistance = maxInt(1, len(corrected)/20)
	}

	type candidate struct {
		index      int
		leftBase   int
		rightBase  int
		prominence float64
	}

	var candidates []candidate
	for i := 1; i < len(corrected)-1; i++ {
		if corrected[i] < corrected[i-1] || corrected[i] < corrected[i+1] {
			continue
		}
		if corrected[i] == corrected[i-1] {
			continue // not the start of a plateau; the earlier index owns it
		}

		peakEnd := i
		for peakEnd+1 < len(corrected) && corrected[peakEnd+1] == corrected[i] {
			peakEnd++
		}
		peakIdx := (i + peakEnd) / 2
		peakVal := corrected[i]

		leftBase := i
		leftMin := peakVal
		for j := i - 1; j >= 0; j-- {
			if corrected[j] > peakVal {
				break
			}
			if corrected[j] < leftMin {
				leftMin = corrected[j]
				leftBase = j
			}
		}

		rightBase := peakEnd
		rightMin := peakVal
		for j := peakEnd + 1; j < len(corrected); j++ {
			if corrected[j] > peakVal {
				break
			}
			if corrected[j] < rightMin {
				rightMin = corrected[j]
				rightBase = j
			}
		}

		base := leftMin
		if rightMin > base {
			base = rightMin
		}
		prominence := peakVal - base
		if prominence < minProminence*profileMax {
			continue
		}

		candidates = append(candidates, candidate{
			index:      peakIdx,
			leftBase:   leftBase,
			rightBase:  rightBase,
			prominence: prominence,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return corrected[candidates[i].index] > corrected[candidates[j].index]
	})

	var accepted []candidate
	for _, c := range candidates {
		tooClose := false
		for _, a := range accepted {
			if absInt(c.index-a.index) < minDistance {
				tooClose = true
				break
			}
		}
		if !tooClose {
			accepted = append(accepted, c)
		}
	}

	sort.Slice(accepted, func(i, j int) bool { return accepted[i].index < accepted[j].index })

	var totalArea float64
	bands := make([]models.GelBand, 0, len(accepted))
	for _, c := range accepted {
		var area float64
		for j := c.leftBase; j < c.rightBase; j++ {
			area += (corrected[j] + corrected[j+1]) / 2
		}
		totalArea += area

		bands = append(bands, models.GelBand{
			Position:         float64(c.index),
			RelativePosition: relativePosition(c.index, len(values)),
			Intensity:        corrected[c.index],
			Area:             area,
			Width:            float64(c.rightBase - c.leftBase),
		})
	}

	if totalArea > 0 {
		for i := range bands {
			bands[i].RelativeQuantity = bands[i].Area / totalArea * 100
		}
	}

	return bands
}

func relativePosition(index, length int) float64 {
	if length <= 1 {
		return 0
	}
	return float64(index) / float64(length-1)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// FitCalibrationCurve fits log10(MW) as a linear function of relative migration distance via
// ordinary least squares — the standard SDS-PAGE calibration convention.
func FitCalibrationCurve(points []models.GelCalibrationPoint) (models.GelCalibrationCurve, error) {
	n := len(points)
	if n < 2 {
		return models.GelCalibrationCurve{}, fmt.Errorf("at least 2 calibration points are required, got %d", n)
	}

	var sumX, sumY, sumXY, sumX2 float64
	for _, p := range points {
		sumX += p.Position
		sumY += p.LogMW
		sumXY += p.Position * p.LogMW
		sumX2 += p.Position * p.Position
	}
	nf := float64(n)

	denom := nf*sumX2 - sumX*sumX
	if denom == 0 {
		return models.GelCalibrationCurve{}, fmt.Errorf("calibration points have no spread in migration distance")
	}

	slope := (nf*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / nf

	meanY := sumY / nf
	var ssRes, ssTot float64
	for _, p := range points {
		predicted := slope*p.Position + intercept
		ssRes += (p.LogMW - predicted) * (p.LogMW - predicted)
		ssTot += (p.LogMW - meanY) * (p.LogMW - meanY)
	}

	rSquared := 1.0
	if ssTot > 0 {
		rSquared = 1 - ssRes/ssTot
	}

	return models.GelCalibrationCurve{
		Slope:     slope,
		Intercept: intercept,
		RSquared:  rSquared,
		Points:    points,
	}, nil
}

// ApplyCalibrationToProfile resolves MolecularWeight for every band in profile using curve.
func ApplyCalibrationToProfile(profile *models.GelLaneProfile, curve models.GelCalibrationCurve) {
	for i := range profile.Bands {
		logMW := curve.Slope*profile.Bands[i].RelativePosition + curve.Intercept
		mw := math.Pow(10, logMW)
		profile.Bands[i].MolecularWeight = &mw
	}
}
