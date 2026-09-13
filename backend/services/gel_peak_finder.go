package services

import (
	"fmt"
	"math"
	"sort"

	"github.com/noatgnu/cauldron-go/backend/models"
)

// AnalysisEngineVersion identifies this file's math behavior. Bump only when a result-affecting change is made, not for unrelated app changes.
const AnalysisEngineVersion = "1.1.0"

// SmoothProfile applies a centered moving average; a window of 0 or 1 disables smoothing.
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

// ComputeBaseline estimates a local background to subtract before peak detection: "rolling-min" (default), "percentile" (5th, more noise-tolerant), or "none". polarity picks which side of each window is background: light-bands tracks the low envelope, dark-bands tracks the high envelope.
func ComputeBaseline(values []float64, method string, polarity string) []float64 {
	baseline := make([]float64, len(values))
	if len(values) == 0 {
		return baseline
	}
	if method == "" {
		method = "rolling-min"
	}
	trackHigh := polarity == "dark-bands"
	if method == "none" {
		if trackHigh {
			maxVal := values[0]
			for _, v := range values {
				if v > maxVal {
					maxVal = v
				}
			}
			for i := range baseline {
				baseline[i] = maxVal
			}
		}
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
			if trackHigh {
				idx = len(windowVals) - 1 - idx
			}
			baseline[i] = windowVals[idx]
		default: // "rolling-min"/"rolling-max"
			if trackHigh {
				baseline[i] = windowVals[len(windowVals)-1]
			} else {
				baseline[i] = windowVals[0]
			}
		}
	}
	return baseline
}

// DetectPolarity infers dark-bands or light-bands from a profile's own mean-vs-median skew.
func DetectPolarity(values []float64) string {
	if len(values) == 0 {
		return "dark-bands"
	}

	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	median := sorted[mid]
	if len(sorted)%2 == 0 {
		median = (sorted[mid-1] + sorted[mid]) / 2
	}

	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	if mean > median {
		return "light-bands"
	}
	return "dark-bands"
}

type peakCandidate struct {
	index      int
	leftBase   int
	rightBase  int
	prominence float64
}

// trimEdgeOutliers drops leading/trailing candidates isolated from the main cluster by a much larger gap than typical; candidates must already be sorted by index.
func trimEdgeOutliers(candidates []peakCandidate) []peakCandidate {
	const isolationRatio = 3.0
	for len(candidates) >= 3 {
		gaps := make([]int, len(candidates)-1)
		for i := 1; i < len(candidates); i++ {
			gaps[i-1] = candidates[i].index - candidates[i-1].index
		}
		sortedGaps := append([]int(nil), gaps...)
		sort.Ints(sortedGaps)
		median := float64(sortedGaps[len(sortedGaps)/2])
		if median <= 0 {
			break
		}

		frontGap := float64(gaps[0])
		backGap := float64(gaps[len(gaps)-1])
		if frontGap > isolationRatio*median {
			candidates = candidates[1:]
			continue
		}
		if backGap > isolationRatio*median {
			candidates = candidates[:len(candidates)-1]
			continue
		}
		break
	}
	return candidates
}

// computeCorrectedProfile applies the same smoothing and baseline subtraction as FindPeaks, shared with manual band synthesis.
func computeCorrectedProfile(values, baseline []float64, params models.GelPeakParams) (corrected []float64, profileMax float64) {
	smoothWindow := params.SmoothingWindow
	if smoothWindow == 0 {
		smoothWindow = 7
	}
	smoothed := SmoothProfile(values, smoothWindow)

	polarity := params.Polarity
	if polarity == "" {
		polarity = DetectPolarity(values)
	}

	corrected = make([]float64, len(smoothed))
	for i := range smoothed {
		var v float64
		if i < len(baseline) {
			if polarity == "light-bands" {
				v = smoothed[i] - baseline[i]
			} else {
				v = baseline[i] - smoothed[i]
			}
		} else {
			v = smoothed[i]
		}
		if v < 0 {
			v = 0
		}
		corrected[i] = v
		if v > profileMax {
			profileMax = v
		}
	}
	return corrected, profileMax
}

// FindPeaks is a dependency-free equivalent of scipy.signal.find_peaks: local maxima -> prominence filter -> min-distance non-max suppression.
func FindPeaks(values, baseline []float64, params models.GelPeakParams) []models.GelBand {
	if len(values) == 0 {
		return nil
	}

	corrected, profileMax := computeCorrectedProfile(values, baseline, params)
	if profileMax == 0 {
		return nil
	}

	minProminence := params.MinProminence
	if minProminence <= 0 {
		minProminence = 0.035
	}
	minDistance := params.MinDistance
	if minDistance <= 0 {
		minDistance = maxInt(1, len(corrected)/130)
	}

	// Bounds how far the prominence base search may travel, so a slow background drift can't fake a large prominence.
	baseSearchCap := 2 * minDistance

	var candidates []peakCandidate
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
		leftLimit := i - baseSearchCap
		for j := i - 1; j >= 0 && j >= leftLimit; j-- {
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
		rightLimit := peakEnd + baseSearchCap
		for j := peakEnd + 1; j < len(corrected) && j <= rightLimit; j++ {
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

		candidates = append(candidates, peakCandidate{
			index:      peakIdx,
			leftBase:   leftBase,
			rightBase:  rightBase,
			prominence: prominence,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return corrected[candidates[i].index] > corrected[candidates[j].index]
	})

	var accepted []peakCandidate
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
	accepted = trimEdgeOutliers(accepted)

	if params.EdgeExclusionFraction > 0 {
		lo := params.EdgeExclusionFraction * float64(len(values)-1)
		hi := float64(len(values)-1) - lo
		filtered := accepted[:0]
		for _, c := range accepted {
			if float64(c.index) >= lo && float64(c.index) <= hi {
				filtered = append(filtered, c)
			}
		}
		accepted = filtered
	}

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

// ApplyBandOverrides removes Excluded bands and adds a manual band per non-excluded override, then recomputes RelativeQuantity.
func ApplyBandOverrides(bands []models.GelBand, overrides []models.GelBandOverride, values, baseline []float64, params models.GelPeakParams) []models.GelBand {
	if len(overrides) == 0 {
		return bands
	}

	excluded := make(map[float64]bool)
	for _, o := range overrides {
		if o.Excluded {
			excluded[o.Position] = true
		}
	}

	kept := make([]models.GelBand, 0, len(bands))
	for _, b := range bands {
		if excluded[b.Position] {
			continue
		}
		kept = append(kept, b)
	}

	minDistance := params.MinDistance
	if minDistance <= 0 {
		minDistance = maxInt(1, len(values)/130)
	}
	var corrected []float64
	for _, o := range overrides {
		if o.Excluded {
			continue
		}
		alreadyCovered := false
		for _, b := range kept {
			if absFloat(b.Position-o.Position) < float64(minDistance) {
				alreadyCovered = true
				break
			}
		}
		if alreadyCovered {
			continue
		}
		if corrected == nil {
			corrected, _ = computeCorrectedProfile(values, baseline, params)
		}
		kept = append(kept, synthesizeManualBand(corrected, len(values), o))
	}

	sort.Slice(kept, func(i, j int) bool { return kept[i].Position < kept[j].Position })

	var totalArea float64
	for _, b := range kept {
		totalArea += b.Area
	}
	if totalArea > 0 {
		for i := range kept {
			kept[i].RelativeQuantity = kept[i].Area / totalArea * 100
		}
	}

	return kept
}

// synthesizeManualBand measures Intensity/Area/Width directly from the corrected profile over [Position-Width/2, Position+Width/2].
func synthesizeManualBand(corrected []float64, profileLength int, o models.GelBandOverride) models.GelBand {
	width := o.Width
	if width <= 0 {
		width = 1
	}
	left := clampInt(int(math.Round(o.Position-width/2)), 0, len(corrected)-1)
	right := clampInt(int(math.Round(o.Position+width/2)), 0, len(corrected)-1)
	if right < left {
		right = left
	}

	intensity := corrected[left]
	var area float64
	for j := left; j < right; j++ {
		area += (corrected[j] + corrected[j+1]) / 2
		if corrected[j+1] > intensity {
			intensity = corrected[j+1]
		}
	}

	return models.GelBand{
		Position:         o.Position,
		RelativePosition: relativePosition(int(math.Round(o.Position)), profileLength),
		Intensity:        intensity,
		Area:             area,
		Width:            float64(right - left),
	}
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
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

// FitCalibrationCurve fits log10(MW) vs. relative migration distance via OLS, the standard SDS-PAGE calibration convention.
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
