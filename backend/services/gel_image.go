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

// LoadGelImageFile decodes a gel scan (PNG, JPEG, or TIFF, detected by content rather than
// extension) into a GelImageBuffer, always promoted to 16-bit grayscale so downstream math
// (per-lane intensity summation) doesn't lose precision for sources that are natively 16-bit,
// which is common for gel/blot scanners.
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
			return nil, 0, fmt.Errorf("this TIFF variant isn't supported for direct load — try Auto-detect, which uses Python's TIFF reader and can normalize it for you (decode error: %w)", err)
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

// toGray16Buffer promotes any decoded image to a 16-bit grayscale buffer. Images that are
// already Gray16 are copied directly (no precision loss); everything else is converted via
// image/color's standard luminance weighting.
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

// SumColumnRange sums pixel intensity across columns [x0,x1) for each row in [y0,y1), returning
// one summed value per row — the raw intensity profile for a lane ROI spanning that rectangle.
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

// EncodePreviewPNG renders an 8-bit, min/max-contrast-stretched PNG suitable for canvas display.
// The original 16-bit buffer is untouched — only this display copy is downsampled.
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

// EncodeGray16PNG losslessly encodes the buffer as a 16-bit grayscale PNG — used to hand the
// working image to the Python auto-detect script without any precision loss.
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
