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
