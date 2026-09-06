package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"

	"github.com/noatgnu/cauldron-go/backend/models"
	"golang.org/x/image/tiff"
)

// GelImageBuffer holds a decoded gel image promoted to 16-bit-per-pixel grayscale, row-major.
type GelImageBuffer struct {
	Width  int
	Height int
	Gray16 []uint16
}

// LoadGelImageFile decodes a gel scan (PNG/JPEG/TIFF, detected by content) into a GelImageBuffer, always promoted to 16-bit grayscale.
func LoadGelImageFile(path string) (*GelImageBuffer, models.GelImageMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, models.GelImageMeta{}, fmt.Errorf("failed to read image file: %w", err)
	}

	img, bitDepth, err := decodeGelImage(data)
	if err != nil {
		return nil, models.GelImageMeta{}, err
	}

	buf := toGray16Buffer(img)

	hash := sha256.Sum256(data)
	meta := models.GelImageMeta{
		Width:       buf.Width,
		Height:      buf.Height,
		BitDepth:    bitDepth,
		SourcePath:  path,
		ImageSHA256: hex.EncodeToString(hash[:]),
	}

	return buf, meta, nil
}

func decodeGelImage(data []byte) (image.Image, int, error) {
	switch {
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, 0, fmt.Errorf("failed to decode PNG: %w", err)
		}
		return img, bitDepthOf(img), nil

	case len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8:
		img, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, 0, fmt.Errorf("failed to decode JPEG: %w", err)
		}
		return img, 8, nil

	case len(data) >= 4 && (bytes.Equal(data[:4], []byte{'I', 'I', 42, 0}) || bytes.Equal(data[:4], []byte{'M', 'M', 0, 42})):
		img, err := tiff.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, 0, fmt.Errorf("this TIFF variant isn't supported for direct load; try Auto-detect, which uses Python's TIFF reader and can normalize it for you (decode error: %w)", err)
		}
		return img, bitDepthOf(img), nil

	default:
		return nil, 0, fmt.Errorf("unrecognized image format (expected PNG, JPEG, or TIFF)")
	}
}

func bitDepthOf(img image.Image) int {
	if _, ok := img.(*image.Gray16); ok {
		return 16
	}
	return 8
}

// toGray16Buffer promotes any decoded image to 16-bit grayscale; Gray16 sources copy directly, others convert via image/color luminance weighting.
func toGray16Buffer(img image.Image) *GelImageBuffer {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	buf := &GelImageBuffer{Width: width, Height: height, Gray16: make([]uint16, width*height)}

	if gray16, ok := img.(*image.Gray16); ok {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				buf.Gray16[y*width+x] = gray16.Gray16At(bounds.Min.X+x, bounds.Min.Y+y).Y
			}
		}
		return buf
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.Gray16Model.Convert(img.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.Gray16)
			buf.Gray16[y*width+x] = c.Y
		}
	}
	return buf
}

func clampInt(v, minV, maxV int) int {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

// SumColumnRange sums pixel intensity across columns [x0,x1) for each row in [y0,y1): the raw intensity profile for a lane ROI.
func (b *GelImageBuffer) SumColumnRange(x0, y0, x1, y1 int) []float64 {
	x0 = clampInt(x0, 0, b.Width)
	x1 = clampInt(x1, 0, b.Width)
	y0 = clampInt(y0, 0, b.Height)
	y1 = clampInt(y1, 0, b.Height)

	if x1 <= x0 || y1 <= y0 {
		return nil
	}

	values := make([]float64, y1-y0)
	for y := y0; y < y1; y++ {
		var sum float64
		rowOffset := y * b.Width
		for x := x0; x < x1; x++ {
			sum += float64(b.Gray16[rowOffset+x])
		}
		values[y-y0] = sum
	}
	return values
}

// FindLaneCenterX returns the centroid of background-deviation signal in a window around [x,x+width), Gaussian-weighted by distance from the box's own center so a stronger neighboring lane can't hijack it.
func (b *GelImageBuffer) FindLaneCenterX(x, y, width, height, searchMargin float64, polarity string) (float64, error) {
	y0 := clampInt(int(y), 0, b.Height)
	y1 := clampInt(int(y+height), 0, b.Height)
	if y1 <= y0 {
		return 0, fmt.Errorf("invalid row range")
	}

	x0 := clampInt(int(x-searchMargin), 0, b.Width)
	x1 := clampInt(int(x+width+searchMargin), 0, b.Width)
	if x1 <= x0 {
		return 0, fmt.Errorf("invalid search window")
	}

	profile := b.SumColumnProfile(x0, y0, x1, y1)

	// Local rolling max (dark-bands) or min (light-bands), not one global reference. A real scan
	// can have an illumination gradient across the search window, which would otherwise bias a
	// single global reference and drag the centroid toward whichever end is naturally brighter.
	reference := rollingExtreme(profile, int(width*1.5), polarity == "light-bands")

	originalCenter := x + width/2
	sigma := width * 0.4

	var weightedSum, totalWeight float64
	for i, v := range profile {
		col := float64(x0 + i)
		dist := col - originalCenter
		distanceWeight := math.Exp(-(dist * dist) / (2 * sigma * sigma))

		signal := reference[i] - v
		if polarity == "light-bands" {
			signal = v - reference[i]
		}
		if signal < 0 {
			signal = 0
		}

		weight := signal * distanceWeight
		weightedSum += weight * col
		totalWeight += weight
	}
	if totalWeight == 0 {
		return x + width/2, nil
	}

	return weightedSum / totalWeight, nil
}

// rollingExtreme returns, per index, the max (or min if useMin) of profile within a centered window. A simple morphological local-background estimate that tracks a smooth gradient.
func rollingExtreme(profile []float64, window int, useMin bool) []float64 {
	if window < 3 {
		window = 3
	}
	n := len(profile)
	out := make([]float64, n)
	half := window / 2
	for i := 0; i < n; i++ {
		lo := i - half
		if lo < 0 {
			lo = 0
		}
		hi := i + half
		if hi >= n {
			hi = n - 1
		}
		best := profile[lo]
		for j := lo + 1; j <= hi; j++ {
			if useMin {
				if profile[j] < best {
					best = profile[j]
				}
			} else if profile[j] > best {
				best = profile[j]
			}
		}
		out[i] = best
	}
	return out
}

// SumColumnProfile sums pixel intensity across rows [y0,y1) for each column in [x0,x1): the horizontal counterpart to SumColumnRange.
func (b *GelImageBuffer) SumColumnProfile(x0, y0, x1, y1 int) []float64 {
	x0 = clampInt(x0, 0, b.Width)
	x1 = clampInt(x1, 0, b.Width)
	y0 = clampInt(y0, 0, b.Height)
	y1 = clampInt(y1, 0, b.Height)

	if x1 <= x0 || y1 <= y0 {
		return nil
	}

	profile := make([]float64, x1-x0)
	for row := y0; row < y1; row++ {
		rowOffset := row * b.Width
		for col := x0; col < x1; col++ {
			profile[col-x0] += float64(b.Gray16[rowOffset+col])
		}
	}
	return profile
}

// EncodePreviewPNG renders an 8-bit, min/max-contrast-stretched PNG for canvas display; the original 16-bit buffer is untouched.
func EncodePreviewPNG(b *GelImageBuffer) ([]byte, error) {
	if len(b.Gray16) == 0 {
		return nil, fmt.Errorf("empty image buffer")
	}

	lo, hi := b.Gray16[0], b.Gray16[0]
	for _, v := range b.Gray16 {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	span := float64(hi) - float64(lo)

	img := image.NewGray(image.Rect(0, 0, b.Width, b.Height))
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			v := b.Gray16[y*b.Width+x]
			var scaled uint8
			if span > 0 {
				scaled = uint8((float64(v-lo) / span) * 255)
			}
			img.SetGray(x, y, color.Gray{Y: scaled})
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, fmt.Errorf("failed to encode preview PNG: %w", err)
	}
	return out.Bytes(), nil
}

// EncodePreviewPNGWithLevels renders an 8-bit PNG using explicit black/white points (each a
// fraction 0-1 of the full 16-bit range) instead of auto min/max stretching, for user-adjustable
// display brightness/contrast that never touches the underlying 16-bit data or any analysis.
func EncodePreviewPNGWithLevels(b *GelImageBuffer, blackPoint, whitePoint float64) ([]byte, error) {
	if len(b.Gray16) == 0 {
		return nil, fmt.Errorf("empty image buffer")
	}
	if whitePoint <= blackPoint {
		return nil, fmt.Errorf("whitePoint must be greater than blackPoint")
	}

	lo := blackPoint * 65535.0
	span := (whitePoint - blackPoint) * 65535.0

	img := image.NewGray(image.Rect(0, 0, b.Width, b.Height))
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			scaled := (float64(b.Gray16[y*b.Width+x]) - lo) / span * 255.0
			if scaled < 0 {
				scaled = 0
			} else if scaled > 255 {
				scaled = 255
			}
			img.SetGray(x, y, color.Gray{Y: uint8(scaled)})
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, fmt.Errorf("failed to encode preview PNG: %w", err)
	}
	return out.Bytes(), nil
}

// EncodeGray16PNG losslessly encodes the buffer as a 16-bit grayscale PNG, for handing off to the Python auto-detect script.
func EncodeGray16PNG(b *GelImageBuffer) ([]byte, error) {
	img := image.NewGray16(image.Rect(0, 0, b.Width, b.Height))
	for y := 0; y < b.Height; y++ {
		for x := 0; x < b.Width; x++ {
			img.SetGray16(x, y, color.Gray16{Y: b.Gray16[y*b.Width+x]})
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, fmt.Errorf("failed to encode 16-bit PNG: %w", err)
	}
	return out.Bytes(), nil
}
