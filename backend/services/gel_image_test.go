package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeTestPNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("failed to encode test PNG: %v", err)
	}
}

func TestLoadGelImageFile_Gray16RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gray16.png")

	src := image.NewGray16(image.Rect(0, 0, 4, 3))
	src.SetGray16(0, 0, color.Gray16{Y: 100})
	src.SetGray16(1, 0, color.Gray16{Y: 40000})
	src.SetGray16(3, 2, color.Gray16{Y: 65535})
	writeTestPNG(t, path, src)

	buf, meta, err := LoadGelImageFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Width != 4 || meta.Height != 3 {
		t.Errorf("dimensions = %dx%d, want 4x3", meta.Width, meta.Height)
	}
	if meta.BitDepth != 16 {
		t.Errorf("bitDepth = %d, want 16", meta.BitDepth)
	}
	if buf.Gray16[0] != 100 {
		t.Errorf("pixel(0,0) = %d, want 100", buf.Gray16[0])
	}
	if buf.Gray16[1] != 40000 {
		t.Errorf("pixel(1,0) = %d, want 40000", buf.Gray16[1])
	}
	if buf.Gray16[2*4+3] != 65535 {
		t.Errorf("pixel(3,2) = %d, want 65535", buf.Gray16[2*4+3])
	}
}

func TestLoadGelImageFile_Gray8Promotion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gray8.png")

	src := image.NewGray(image.Rect(0, 0, 2, 2))
	src.SetGray(0, 0, color.Gray{Y: 0})
	src.SetGray(1, 1, color.Gray{Y: 255})
	writeTestPNG(t, path, src)

	buf, meta, err := LoadGelImageFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.BitDepth != 8 {
		t.Errorf("bitDepth = %d, want 8", meta.BitDepth)
	}
	if buf.Gray16[3] == 0 {
		t.Errorf("promoted white pixel should not be 0, got %d", buf.Gray16[3])
	}
}

func TestLoadGelImageFile_ImageSHA256(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hashed.png")

	src := image.NewGray16(image.Rect(0, 0, 3, 3))
	src.SetGray16(1, 1, color.Gray16{Y: 12345})
	writeTestPNG(t, path, src)

	fileBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture file: %v", err)
	}
	wantHash := sha256.Sum256(fileBytes)
	wantHashHex := hex.EncodeToString(wantHash[:])

	_, meta, err := LoadGelImageFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.ImageSHA256 != wantHashHex {
		t.Errorf("ImageSHA256 = %q, want %q", meta.ImageSHA256, wantHashHex)
	}
}

func TestLoadGelImageFile_UnrecognizedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "not-an-image.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, _, err := LoadGelImageFile(path)
	if err == nil {
		t.Fatal("expected an error for an unrecognized format")
	}
}

func TestSumColumnRange(t *testing.T) {
	buf := &GelImageBuffer{
		Width:  3,
		Height: 2,
		Gray16: []uint16{1, 2, 3, 4, 5, 6},
	}

	values := buf.SumColumnRange(0, 0, 2, 2)
	if len(values) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(values))
	}
	if values[0] != 3 { // row0: cols 0,1 = 1+2
		t.Errorf("row 0 sum = %v, want 3", values[0])
	}
	if values[1] != 9 { // row1: cols 0,1 = 4+5
		t.Errorf("row 1 sum = %v, want 9", values[1])
	}
}

func TestSumColumnRange_ClampsOutOfBounds(t *testing.T) {
	buf := &GelImageBuffer{Width: 2, Height: 2, Gray16: []uint16{1, 2, 3, 4}}
	values := buf.SumColumnRange(-5, -5, 100, 100)
	if len(values) != 2 {
		t.Fatalf("expected clamped range of 2 rows, got %d", len(values))
	}
}

func TestSumColumnProfile(t *testing.T) {
	buf := &GelImageBuffer{
		Width:  3,
		Height: 2,
		Gray16: []uint16{1, 2, 3, 4, 5, 6},
	}

	profile := buf.SumColumnProfile(0, 0, 3, 2)
	want := []float64{5, 7, 9} // col0: 1+4, col1: 2+5, col2: 3+6
	if len(profile) != len(want) {
		t.Fatalf("expected %d columns, got %d", len(want), len(profile))
	}
	for i := range want {
		if profile[i] != want[i] {
			t.Errorf("column %d = %v, want %v", i, profile[i], want[i])
		}
	}
}

func buildOffsetLaneBuffer(width, height, bandStart, bandEnd int) *GelImageBuffer {
	buf := &GelImageBuffer{Width: width, Height: height, Gray16: make([]uint16, width*height)}
	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			v := uint16(60000)
			if col >= bandStart && col < bandEnd {
				v = 1000
			}
			buf.Gray16[row*width+col] = v
		}
	}
	return buf
}

func TestFindLaneCenterX_MovesTowardOffsetSignal(t *testing.T) {
	// True lane is a dark band at columns [25,35); the rough rectangle is offset left at [20,30),
	// partially overlapping it. Distance-weighting deliberately damps full snap-to-band correction
	// (needed to resist a stronger neighboring lane elsewhere), so this checks partial, bounded
	// movement toward the true centroid (29.5) rather than an exact snap.
	buf := buildOffsetLaneBuffer(60, 4, 25, 35)

	centerX, err := buf.FindLaneCenterX(20, 0, 10, 4, 5, "dark-bands")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if centerX <= 25 || centerX >= 29.5 {
		t.Errorf("centerX = %v, want strictly between original center 25 and true centroid 29.5", centerX)
	}
}

func TestFindLaneCenterX_DoesNotGetPulledIntoStrongerNeighborLane(t *testing.T) {
	// Regression test for a real reported bug: a strong lane [0,60) sits right next to a much
	// fainter lane [80,140) across a small gap. A rough box drawn on the faint lane, reaching
	// slightly into the strong lane's edge via the search margin, must stay centered within the
	// faint lane's own true span rather than getting pulled toward the stronger neighbor.
	const width, height = 200, 4
	buf := &GelImageBuffer{Width: width, Height: height, Gray16: make([]uint16, width*height)}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			v := uint16(60000)
			if x < 60 {
				v = 30000 // strong neighboring lane
			} else if x >= 80 && x < 140 {
				v = 58000 // faint target lane
			}
			buf.Gray16[y*width+x] = v
		}
	}

	centerX, err := buf.FindLaneCenterX(70, 0, 60, height, 30, "dark-bands")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if centerX < 80 || centerX >= 140 {
		t.Errorf("centerX = %v, want within the faint lane's true span [80,140), not pulled into the strong neighbor", centerX)
	}
}

func TestFindLaneCenterX_NoSignalFallsBackToCurrentCenter(t *testing.T) {
	buf := &GelImageBuffer{Width: 40, Height: 4, Gray16: make([]uint16, 40*4)}
	for i := range buf.Gray16 {
		buf.Gray16[i] = 50000 // uniform, no band anywhere
	}

	centerX, err := buf.FindLaneCenterX(10, 0, 10, 4, 5, "dark-bands")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if centerX != 15 { // 10 + 10/2, unchanged
		t.Errorf("centerX = %v, want 15 (unchanged center)", centerX)
	}
}

// TestFindLaneCenterX_RealScannedGel uses a real scanned fluorescent gel crop (light-bands, real
// sensor noise, real background drift) instead of a synthetic buffer. Cropped from 14_NEB.tif in
// the GelGenie quantitation_ladder_gels dataset (Kotidis et al., Nature Communications 2025,
// CC-BY-4.0, https://doi.org/10.5281/zenodo.14641949). It contains a strong lane (~columns 15-80
// in crop coordinates) and a much fainter lane (~columns 167-186) across a gap.
func TestFindLaneCenterX_RealScannedGel(t *testing.T) {
	buf, meta, err := LoadGelImageFile(filepath.Join("testdata", "gelgenie_14_neb_crop.tif"))
	if err != nil {
		t.Fatalf("failed to load real gel fixture: %v", err)
	}
	if meta.Width != 220 || meta.Height != 120 {
		t.Fatalf("unexpected fixture dimensions %dx%d", meta.Width, meta.Height)
	}

	for _, tc := range []struct {
		name   string
		roughX float64
	}{
		{"box_close_to_faint_lane", 150},
		{"box_slightly_left", 155},
		{"box_slightly_right", 160},
	} {
		t.Run(tc.name, func(t *testing.T) {
			centerX, err := buf.FindLaneCenterX(tc.roughX, 10, 40, 100, 20, "light-bands")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// True faint-lane peak is ~173; must land within the faint lane's real span (~167-186)
			// and, crucially, nowhere near the much stronger neighboring lane (~15-80).
			if centerX < 160 || centerX > 190 {
				t.Errorf("centerX = %v, want within the real faint lane's span [160,190]", centerX)
			}
		})
	}
}

func TestEncodePreviewPNG_ProducesValidPNG(t *testing.T) {
	buf := &GelImageBuffer{Width: 2, Height: 2, Gray16: []uint16{0, 30000, 60000, 65535}}
	data, err := EncodePreviewPNG(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output isn't a valid PNG: %v", err)
	}
	if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
		t.Errorf("decoded dims = %dx%d, want 2x2", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

func TestEncodePreviewPNGWithLevels_ClampsAndScales(t *testing.T) {
	// Values chosen so black/white points at 0.25/0.75 of the 16-bit range map cleanly: anything
	// at or below the black point clips to 0, at or above the white point clips to 255, and the
	// midpoint between them maps to roughly the middle of the 8-bit range.
	blackPoint, whitePoint := 0.25, 0.75
	lo := uint16(blackPoint * 65535)
	hi := uint16(whitePoint * 65535)
	mid := uint16((float64(lo) + float64(hi)) / 2)

	buf := &GelImageBuffer{Width: 4, Height: 1, Gray16: []uint16{0, lo, mid, 65535}}
	data, err := EncodePreviewPNGWithLevels(buf, blackPoint, whitePoint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output isn't a valid PNG: %v", err)
	}
	gray, ok := img.(*image.Gray)
	if !ok {
		t.Fatalf("expected *image.Gray, got %T", img)
	}

	if v := gray.GrayAt(0, 0).Y; v != 0 {
		t.Errorf("below black point: got %d, want 0", v)
	}
	if v := gray.GrayAt(1, 0).Y; v != 0 {
		t.Errorf("at black point: got %d, want 0", v)
	}
	if v := gray.GrayAt(3, 0).Y; v != 255 {
		t.Errorf("above white point: got %d, want 255", v)
	}
	if v := gray.GrayAt(2, 0).Y; v < 100 || v > 155 {
		t.Errorf("midpoint: got %d, want roughly 128", v)
	}
}

func TestEncodePreviewPNGWithLevels_RejectsInvalidRange(t *testing.T) {
	buf := &GelImageBuffer{Width: 1, Height: 1, Gray16: []uint16{100}}
	if _, err := EncodePreviewPNGWithLevels(buf, 0.5, 0.5); err == nil {
		t.Error("expected an error when whitePoint == blackPoint")
	}
	if _, err := EncodePreviewPNGWithLevels(buf, 0.7, 0.3); err == nil {
		t.Error("expected an error when whitePoint < blackPoint")
	}
}

func TestEncodeGray16PNG_LosslessRoundTrip(t *testing.T) {
	buf := &GelImageBuffer{Width: 2, Height: 1, Gray16: []uint16{12345, 65535}}
	data, err := EncodeGray16PNG(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.png")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	decoded, _, err := LoadGelImageFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded.Gray16[0] != 12345 || decoded.Gray16[1] != 65535 {
		t.Errorf("round-tripped pixels = %v, want [12345 65535]", decoded.Gray16)
	}
}
