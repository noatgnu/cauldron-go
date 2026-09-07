package services

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// tiffTagSpec describes one IFD entry to write via buildMinimalTIFF; ascii values are encoded as
// TIFF type 2 (NUL-terminated), matching how real gel-scanner software stores text fields.
type tiffTagSpec struct {
	id    uint16
	ascii string
}

// buildMinimalTIFF hand-assembles a single-IFD, little-endian TIFF with the given ASCII tags, so
// the IFD parser can be tested deterministically without depending on a real scanner file.
func buildMinimalTIFF(t *testing.T, tags []tiffTagSpec) []byte {
	t.Helper()

	const ifdOffset = 8
	entryCount := len(tags)
	dataStart := ifdOffset + 2 + entryCount*12 + 4

	var out bytes.Buffer
	out.WriteString("II")
	binary.Write(&out, binary.LittleEndian, uint16(42))
	binary.Write(&out, binary.LittleEndian, uint32(ifdOffset))

	var extraData bytes.Buffer
	binary.Write(&out, binary.LittleEndian, uint16(entryCount))
	for _, tag := range tags {
		valueBytes := []byte(tag.ascii + "\x00")
		binary.Write(&out, binary.LittleEndian, tag.id)
		binary.Write(&out, binary.LittleEndian, uint16(2)) // ASCII
		binary.Write(&out, binary.LittleEndian, uint32(len(valueBytes)))

		if len(valueBytes) <= 4 {
			padded := make([]byte, 4)
			copy(padded, valueBytes)
			out.Write(padded)
		} else {
			binary.Write(&out, binary.LittleEndian, uint32(dataStart+extraData.Len()))
			extraData.Write(valueBytes)
		}
	}
	binary.Write(&out, binary.LittleEndian, uint32(0)) // no next IFD
	out.Write(extraData.Bytes())

	return out.Bytes()
}

func TestExtractRawImageMetadata_TIFFInlineAndOutOfLineTags(t *testing.T) {
	data := buildMinimalTIFF(t, []tiffTagSpec{
		{id: 305, ascii: "Test Software v1.0"},            // Software, out-of-line (>4 bytes)
		{id: 65000, ascii: "<Vendor Blob Brightness=1/>"}, // unrecognized private tag, out-of-line
	})

	path := filepath.Join(t.TempDir(), "minimal.tif")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write test TIFF: %v", err)
	}

	metadata, err := ExtractRawImageMetadata(path)
	if err != nil {
		t.Fatalf("ExtractRawImageMetadata: %v", err)
	}

	if metadata["Software"] != "Test Software v1.0" {
		t.Errorf("Software = %q, want %q", metadata["Software"], "Test Software v1.0")
	}
	if metadata["Tag 65000"] != "<Vendor Blob Brightness=1/>" {
		t.Errorf("Tag 65000 = %q, want %q", metadata["Tag 65000"], "<Vendor Blob Brightness=1/>")
	}
}

func TestExtractRawImageMetadata_RealScannedTIFF(t *testing.T) {
	metadata, err := ExtractRawImageMetadata("testdata/gelgenie_14_neb_crop.tif")
	if err != nil {
		t.Fatalf("ExtractRawImageMetadata: %v", err)
	}

	if metadata["Compression"] == "" {
		t.Error("expected a Compression tag to be extracted from the real scanned TIFF fixture")
	}
}

// writePNGWithChunks re-serializes a PNG, inserting the given raw chunks (already length+type+data+crc encoded) right after IHDR.
func writePNGWithChunks(t *testing.T, path string, img image.Image, chunks [][]byte) {
	t.Helper()

	var base bytes.Buffer
	if err := png.Encode(&base, img); err != nil {
		t.Fatalf("failed to encode base PNG: %v", err)
	}
	raw := base.Bytes()

	ihdrLength := binary.BigEndian.Uint32(raw[8:12])
	ihdrEnd := 8 + 8 + int(ihdrLength) + 4 // signature + length/type + data + crc

	var out bytes.Buffer
	out.Write(raw[:ihdrEnd])
	for _, chunk := range chunks {
		out.Write(chunk)
	}
	out.Write(raw[ihdrEnd:])

	if err := os.WriteFile(path, out.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write test PNG: %v", err)
	}
}

// buildPNGChunk assembles one raw PNG chunk (length + type + data + CRC over type+data).
func buildPNGChunk(chunkType string, data []byte) []byte {
	var chunk bytes.Buffer
	binary.Write(&chunk, binary.BigEndian, uint32(len(data)))
	chunk.WriteString(chunkType)
	chunk.Write(data)
	binary.Write(&chunk, binary.BigEndian, crc32.ChecksumIEEE(append([]byte(chunkType), data...)))
	return chunk.Bytes()
}

func TestExtractRawImageMetadata_PNGTextAndPhysChunks(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetGray(x, y, color.Gray{Y: 128})
		}
	}

	tEXtData := append([]byte("Comment\x00"), []byte("hand-drawn gel note")...)

	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	zw.Write([]byte("compressed gel note"))
	zw.Close()
	zTXtData := append(append([]byte("Description\x00"), byte(0)), compressed.Bytes()...)

	physData := make([]byte, 9)
	binary.BigEndian.PutUint32(physData[0:4], 3780) // ~96 DPI in pixels/meter
	binary.BigEndian.PutUint32(physData[4:8], 3780)
	physData[8] = 1 // meter

	chunks := [][]byte{
		buildPNGChunk("tEXt", tEXtData),
		buildPNGChunk("zTXt", zTXtData),
		buildPNGChunk("pHYs", physData),
	}

	path := filepath.Join(t.TempDir(), "with-chunks.png")
	writePNGWithChunks(t, path, img, chunks)

	metadata, err := ExtractRawImageMetadata(path)
	if err != nil {
		t.Fatalf("ExtractRawImageMetadata: %v", err)
	}

	if metadata["Comment"] != "hand-drawn gel note" {
		t.Errorf("Comment = %q, want %q", metadata["Comment"], "hand-drawn gel note")
	}
	if metadata["Description"] != "compressed gel note" {
		t.Errorf("Description = %q, want %q", metadata["Description"], "compressed gel note")
	}
	if metadata["PhysicalPixelDimensions"] != "3780 x 3780 pixels per meter" {
		t.Errorf("PhysicalPixelDimensions = %q, want %q", metadata["PhysicalPixelDimensions"], "3780 x 3780 pixels per meter")
	}
}
