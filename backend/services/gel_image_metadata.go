package services

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// tiffTagNames maps well-known baseline TIFF tag IDs to display names; unrecognized (including
// vendor-private) tags fall back to "Tag <id>" so nothing is silently dropped.
var tiffTagNames = map[uint16]string{
	256: "ImageWidth", 257: "ImageLength", 258: "BitsPerSample", 259: "Compression",
	262: "PhotometricInterpretation", 269: "DocumentName", 270: "ImageDescription",
	271: "Make", 272: "Model", 273: "StripOffsets", 274: "Orientation",
	277: "SamplesPerPixel", 278: "RowsPerStrip", 279: "StripByteCounts",
	282: "XResolution", 283: "YResolution", 284: "PlanarConfiguration",
	296: "ResolutionUnit", 297: "PageNumber", 305: "Software", 306: "DateTime",
	315: "Artist", 317: "Predictor", 339: "SampleFormat", 33432: "Copyright",
}

// maxRawMetadataArrayItems caps how many values of a multi-valued tag/field are rendered, so a
// 171-entry StripOffsets array doesn't flood the display; the total count is always shown too.
const maxRawMetadataArrayItems = 8

// ExtractRawImageMetadata reads unparsed, vendor-agnostic metadata directly from the image file
// (TIFF IFD tags or PNG text/physical-dimension chunks) for display as an audit trail. Values are
// shown as the file actually encodes them, including unrecognized private tags, rather than
// normalized into a fixed schema that would need per-vendor maintenance to stay accurate.
func ExtractRawImageMetadata(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	switch {
	case len(data) >= 4 && (bytes.Equal(data[:4], []byte{'I', 'I', 42, 0}) || bytes.Equal(data[:4], []byte{'M', 'M', 0, 42})):
		return extractTIFFMetadata(data)
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		return extractPNGMetadata(data), nil
	default:
		return map[string]string{}, nil
	}
}

func extractTIFFMetadata(data []byte) (map[string]string, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("TIFF file too small")
	}

	var order binary.ByteOrder = binary.LittleEndian
	if data[0] == 'M' {
		order = binary.BigEndian
	}

	ifdOffset := order.Uint32(data[4:8])
	if int(ifdOffset)+2 > len(data) {
		return nil, fmt.Errorf("invalid TIFF IFD offset")
	}

	numEntries := int(order.Uint16(data[ifdOffset : ifdOffset+2]))
	entriesStart := int(ifdOffset) + 2

	metadata := make(map[string]string, numEntries)
	for i := 0; i < numEntries; i++ {
		entryOffset := entriesStart + i*12
		if entryOffset+12 > len(data) {
			break
		}
		entry := data[entryOffset : entryOffset+12]

		tagID := order.Uint16(entry[0:2])
		fieldType := order.Uint16(entry[2:4])
		count := order.Uint32(entry[4:8])

		value, err := formatTIFFTagValue(data, order, fieldType, count, entry[8:12])
		if err != nil {
			continue
		}

		name, ok := tiffTagNames[tagID]
		if !ok {
			name = fmt.Sprintf("Tag %d", tagID)
		}
		metadata[name] = value
	}

	return metadata, nil
}

// tiffTypeSize returns the byte size of one value of a TIFF field type, and false for an unsupported type.
func tiffTypeSize(fieldType uint16) (int, bool) {
	switch fieldType {
	case 1, 2, 6, 7: // BYTE, ASCII, SBYTE, UNDEFINED
		return 1, true
	case 3, 8: // SHORT, SSHORT
		return 2, true
	case 4, 9, 11: // LONG, SLONG, FLOAT
		return 4, true
	case 5, 10, 12: // RATIONAL, SRATIONAL, DOUBLE
		return 8, true
	default:
		return 0, false
	}
}

// formatTIFFTagValue decodes one IFD entry's value(s) into a human-readable string, resolving the
// out-of-line offset for values larger than 4 bytes.
func formatTIFFTagValue(data []byte, order binary.ByteOrder, fieldType uint16, count uint32, inlineValue []byte) (string, error) {
	itemSize, ok := tiffTypeSize(fieldType)
	if !ok {
		return "", fmt.Errorf("unsupported TIFF field type %d", fieldType)
	}

	totalSize := itemSize * int(count)
	var valueBytes []byte
	if totalSize <= 4 {
		valueBytes = inlineValue[:totalSize]
	} else {
		offset := int(order.Uint32(inlineValue))
		if offset+totalSize > len(data) || offset < 0 {
			return "", fmt.Errorf("TIFF value offset out of range")
		}
		valueBytes = data[offset : offset+totalSize]
	}

	if fieldType == 2 { // ASCII: NUL-terminated (or padded) string
		return strings.TrimRight(string(valueBytes), "\x00"), nil
	}

	items := make([]string, 0, count)
	for i := uint32(0); i < count && int(i) < maxRawMetadataArrayItems; i++ {
		chunk := valueBytes[int(i)*itemSize : int(i+1)*itemSize]
		items = append(items, formatTIFFScalar(order, fieldType, chunk))
	}
	joined := strings.Join(items, ", ")
	if count > maxRawMetadataArrayItems {
		joined += fmt.Sprintf(", ... (%d total)", count)
	}
	return joined, nil
}

func formatTIFFScalar(order binary.ByteOrder, fieldType uint16, chunk []byte) string {
	switch fieldType {
	case 1, 6: // BYTE, SBYTE
		return strconv.Itoa(int(chunk[0]))
	case 3: // SHORT
		return strconv.Itoa(int(order.Uint16(chunk)))
	case 8: // SSHORT
		return strconv.Itoa(int(int16(order.Uint16(chunk))))
	case 4: // LONG
		return strconv.FormatUint(uint64(order.Uint32(chunk)), 10)
	case 9: // SLONG
		return strconv.Itoa(int(int32(order.Uint32(chunk))))
	case 5: // RATIONAL
		return fmt.Sprintf("%d/%d", order.Uint32(chunk[0:4]), order.Uint32(chunk[4:8]))
	case 10: // SRATIONAL
		return fmt.Sprintf("%d/%d", int32(order.Uint32(chunk[0:4])), int32(order.Uint32(chunk[4:8])))
	case 11: // FLOAT
		return strconv.FormatFloat(float64(order.Uint32(chunk)), 'g', -1, 32)
	case 7: // UNDEFINED
		return strconv.Itoa(int(chunk[0]))
	default:
		return ""
	}
}

// extractPNGMetadata reads tEXt/zTXt/iTXt (arbitrary key-value text) and pHYs (physical pixel
// density) chunks; malformed chunks are skipped rather than failing the whole read.
func extractPNGMetadata(data []byte) map[string]string {
	metadata := make(map[string]string)
	offset := 8 // past the 8-byte PNG signature

	for offset+8 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		chunkType := string(data[offset+4 : offset+8])
		dataStart := offset + 8
		dataEnd := dataStart + length
		if length < 0 || dataEnd+4 > len(data) {
			break
		}
		chunkData := data[dataStart:dataEnd]

		switch chunkType {
		case "tEXt":
			if key, text, ok := splitNulSeparated(chunkData); ok {
				metadata[key] = text
			}
		case "zTXt":
			// keyword \0 compressionMethod(1 byte) compressedData. splitNulSeparated only
			// strips the keyword's NUL, so the leading compression-method byte must be dropped too.
			if key, rest, ok := splitNulSeparated(chunkData); ok && len(rest) > 1 {
				if text, err := zlibDecompress([]byte(rest[1:])); err == nil {
					metadata[key] = text
				}
			}
		case "iTXt":
			if key, text, ok := parseITXt(chunkData); ok {
				metadata[key] = text
			}
		case "pHYs":
			if len(chunkData) == 9 {
				ppuX := binary.BigEndian.Uint32(chunkData[0:4])
				ppuY := binary.BigEndian.Uint32(chunkData[4:8])
				unit := "unspecified"
				if chunkData[8] == 1 {
					unit = "meter"
				}
				metadata["PhysicalPixelDimensions"] = fmt.Sprintf("%d x %d pixels per %s", ppuX, ppuY, unit)
			}
		case "tIME":
			if len(chunkData) == 7 {
				year := binary.BigEndian.Uint16(chunkData[0:2])
				metadata["LastModified"] = fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
					year, chunkData[2], chunkData[3], chunkData[4], chunkData[5], chunkData[6])
			}
		}

		offset = dataEnd + 4 // skip the trailing 4-byte CRC
	}

	return metadata
}

// splitNulSeparated splits a tEXt/zTXt chunk's payload into its keyword and text, on the single NUL separator.
func splitNulSeparated(data []byte) (key string, text string, ok bool) {
	idx := bytes.IndexByte(data, 0)
	if idx < 0 {
		return "", "", false
	}
	return string(data[:idx]), string(data[idx+1:]), true
}

// parseITXt parses an iTXt chunk: keyword\0 compressionFlag compressionMethod languageTag\0 translatedKeyword\0 text.
func parseITXt(data []byte) (key string, text string, ok bool) {
	idx := bytes.IndexByte(data, 0)
	if idx < 0 || idx+2 > len(data) {
		return "", "", false
	}
	key = string(data[:idx])
	compressed := data[idx+1] == 1
	rest := data[idx+3:] // skip compression flag + compression method

	idx2 := bytes.IndexByte(rest, 0)
	if idx2 < 0 {
		return "", "", false
	}
	rest = rest[idx2+1:] // skip language tag

	idx3 := bytes.IndexByte(rest, 0)
	if idx3 < 0 {
		return "", "", false
	}
	textBytes := rest[idx3+1:] // skip translated keyword

	if compressed {
		decompressed, err := zlibDecompress(textBytes)
		if err != nil {
			return "", "", false
		}
		return key, decompressed, true
	}
	return key, string(textBytes), true
}

func zlibDecompress(data []byte) (string, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
