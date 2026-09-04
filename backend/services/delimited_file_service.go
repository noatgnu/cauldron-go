package services

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

type openDelimitedHandle struct {
	file      *os.File
	delimiter rune
	header    []string
	index     []int64
	fileSize  int64
}

// DelimitedFileService opens CSV/TSV files via a byte-offset index built once on open, never buffering a whole file into memory.
type DelimitedFileService struct {
	mu       sync.Mutex
	handles  map[string]*openDelimitedHandle
	progress *ProgressNotifier
}

func NewDelimitedFileService(progress *ProgressNotifier) *DelimitedFileService {
	return &DelimitedFileService{
		handles:  make(map[string]*openDelimitedHandle),
		progress: progress,
	}
}

func delimiterFor(path string) rune {
	if strings.ToLower(filepath.Ext(path)) == ".tsv" {
		return '\t'
	}
	return ','
}

// newDelimitedReader builds a csv.Reader over r, tolerating ragged rows.
func newDelimitedReader(r io.Reader, delimiter rune) *csv.Reader {
	reader := csv.NewReader(r)
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	return reader
}

// skipBOM advances past a leading UTF-8 byte-order mark, if present, and returns the number of bytes skipped.
func skipBOM(f *os.File) (int64, error) {
	buf := make([]byte, 3)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if n == 3 && bytes.Equal(buf, utf8BOM) {
		return 3, nil
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	return 0, nil
}

func (s *DelimitedFileService) getOrOpen(path string) (*openDelimitedHandle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if h, ok := s.handles[path]; ok {
		return h, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s (check the network drive is still connected): %w", path, err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to stat %s: %w", path, err)
	}

	bomLen, err := skipBOM(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to read %s (check the network drive is still connected): %w", path, err)
	}

	delimiter := delimiterFor(path)
	reader := newDelimitedReader(f, delimiter)

	header, err := reader.Read()
	if err == io.EOF {
		h := &openDelimitedHandle{file: f, delimiter: delimiter, header: nil, index: nil, fileSize: stat.Size()}
		s.handles[path] = h
		return h, nil
	}
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to read header from %s: %w", path, err)
	}

	index := []int64{bomLen + reader.InputOffset()}
	for {
		if _, err := reader.Read(); err != nil {
			if err == io.EOF {
				break
			}
			f.Close()
			return nil, fmt.Errorf("failed to index %s (check the network drive is still connected): %w", path, err)
		}
		index = append(index, bomLen+reader.InputOffset())
	}
	index = index[:len(index)-1]

	h := &openDelimitedHandle{
		file:      f,
		delimiter: delimiter,
		header:    header,
		index:     index,
		fileSize:  stat.Size(),
	}
	s.handles[path] = h
	return h, nil
}

// OpenFile opens (or reuses an already-open handle for) path and returns its header and row count.
func (s *DelimitedFileService) OpenFile(path string) (*DataFileInfo, error) {
	h, err := s.getOrOpen(path)
	if err != nil {
		return nil, err
	}

	columns := make([]DataColumn, len(h.header))
	for i, name := range h.header {
		columns[i] = DataColumn{Name: name}
	}

	return &DataFileInfo{
		Path:     path,
		Columns:  columns,
		NumRows:  int64(len(h.index)),
		FileSize: h.fileSize,
	}, nil
}

// ReadPage seeks to the indexed byte offset for offset and reads up to limit rows from a fresh reader positioned there.
func (s *DelimitedFileService) ReadPage(path string, offset, limit int) ([]map[string]interface{}, error) {
	h, err := s.getOrOpen(path)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if offset < 0 || offset >= len(h.index) || limit <= 0 {
		return []map[string]interface{}{}, nil
	}

	if _, err := h.file.Seek(h.index[offset], io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to row %d in %s (check the network drive is still connected): %w", offset, path, err)
	}

	reader := newDelimitedReader(h.file, h.delimiter)
	end := offset + limit
	if end > len(h.index) {
		end = len(h.index)
	}

	result := make([]map[string]interface{}, 0, end-offset)
	for i := offset; i < end; i++ {
		record, err := reader.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to read row %d from %s (check the network drive is still connected): %w", i, path, err)
		}
		result = append(result, rowSliceToMap(record, h.header))
	}
	return result, nil
}

func rowSliceToMap(record []string, header []string) map[string]interface{} {
	m := make(map[string]interface{}, len(header))
	for i, name := range header {
		if i < len(record) {
			m[name] = record[i]
		} else {
			m[name] = ""
		}
	}
	return m
}

// ExportToCSV streams rows [offset, offset+limit) through a CSV writer, using its own reader independent of any cached browsing handle for path. offset<=0 starts from the first row; limit<=0 means no upper bound (export to end of file) — this is NOT the same "return nothing" convention ReadPage's limit<=0 uses.
func (s *DelimitedFileService) ExportToCSV(path, outputPath string, columns []string, delimiter rune, offset, limit int) error {
	if offset < 0 {
		offset = 0
	}

	h, err := s.getOrOpen(path)
	if err != nil {
		return err
	}

	// Copy out from under a brief lock so the (potentially long) export below never blocks concurrent ReadPage calls; not covering a real race since CloseFile never mutates a live handle's slices in place.
	s.mu.Lock()
	header := append([]string(nil), h.header...)
	index := append([]int64(nil), h.index...)
	sourceDelimiter := h.delimiter
	s.mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s (check the network drive is still connected): %w", path, err)
	}
	defer f.Close()

	selected := columns
	if len(selected) == 0 {
		selected = header
	}
	colIndex := make(map[string]int, len(header))
	for i, name := range header {
		colIndex[name] = i
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", outputPath, err)
	}
	defer out.Close()

	w := csv.NewWriter(out)
	w.Comma = delimiter
	if err := w.Write(selected); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	s.progress.EmitStart(ProgressTypeGeneric, "table-export", fmt.Sprintf("Exporting %s...", filepath.Base(path)))

	if offset >= len(index) {
		w.Flush()
		s.progress.EmitComplete(ProgressTypeGeneric, "table-export", fmt.Sprintf("Exported 0 rows to %s", filepath.Base(outputPath)))
		return nil
	}

	if _, err := f.Seek(index[offset], io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to row %d in %s (check the network drive is still connected): %w", offset, path, err)
	}
	reader := newDelimitedReader(f, sourceDelimiter)

	total := int64(len(index) - offset)
	if limit > 0 && int64(limit) < total {
		total = int64(limit)
	}

	var written int64
	const batchSize = 1000
	record := make([]string, len(selected))

	for {
		row, readErr := reader.Read()
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			s.progress.EmitError(ProgressTypeGeneric, "table-export", "Export failed", readErr.Error())
			return fmt.Errorf("failed to read rows from %s (check the network drive is still connected): %w", path, readErr)
		}

		for j, name := range selected {
			record[j] = ""
			if idx, ok := colIndex[name]; ok && idx < len(row) {
				record[j] = row[idx]
			}
		}
		if err := w.Write(record); err != nil {
			s.progress.EmitError(ProgressTypeGeneric, "table-export", "Export failed", err.Error())
			return fmt.Errorf("failed to write row: %w", err)
		}

		written++
		if written%batchSize == 0 || (total > 0 && written == total) {
			pct := float64(written) / float64(total) * 100
			if total <= 0 {
				pct = 0
			}
			s.progress.EmitProgress(ProgressTypeGeneric, "table-export", fmt.Sprintf("Exported %d/%d rows", written, total), pct)
		}
		if limit > 0 && written >= int64(limit) {
			break
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		s.progress.EmitError(ProgressTypeGeneric, "table-export", "Export failed", err.Error())
		return fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	s.progress.EmitComplete(ProgressTypeGeneric, "table-export", fmt.Sprintf("Exported %d rows to %s", written, filepath.Base(outputPath)))
	return nil
}

// CloseFile releases the cached handle for path, if one is open.
func (s *DelimitedFileService) CloseFile(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	h, ok := s.handles[path]
	if !ok {
		return nil
	}
	delete(s.handles, path)
	return h.file.Close()
}
