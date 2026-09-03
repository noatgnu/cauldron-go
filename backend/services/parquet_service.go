package services

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/parquet-go/parquet-go"
)

// ParquetColumn describes one leaf column of a Parquet file's schema.
type ParquetColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ParquetFileInfo summarizes a Parquet file's schema and size, read from the file footer only.
type ParquetFileInfo struct {
	Path         string          `json:"path"`
	Columns      []ParquetColumn `json:"columns"`
	NumRows      int64           `json:"numRows"`
	NumRowGroups int             `json:"numRowGroups"`
	FileSize     int64           `json:"fileSize"`
}

type openParquetHandle struct {
	file      *os.File
	pf        *parquet.File
	reader    *parquet.GenericReader[any]
	rowGroups int
}

// ParquetService opens Parquet files via io.ReaderAt-based seeking, never buffering a whole file into memory.
type ParquetService struct {
	mu       sync.Mutex
	handles  map[string]*openParquetHandle
	progress *ProgressNotifier
}

func NewParquetService(progress *ProgressNotifier) *ParquetService {
	return &ParquetService{
		handles:  make(map[string]*openParquetHandle),
		progress: progress,
	}
}

func (s *ParquetService) getOrOpen(path string) (*openParquetHandle, error) {
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

	pf, err := parquet.OpenFile(f, stat.Size())
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to read parquet metadata from %s: %w", path, err)
	}

	h := &openParquetHandle{
		file:      f,
		pf:        pf,
		reader:    parquet.NewGenericReader[any](pf),
		rowGroups: len(pf.RowGroups()),
	}
	s.handles[path] = h
	return h, nil
}

func columnNamesOf(schema *parquet.Schema) []string {
	paths := schema.Columns()
	names := make([]string, len(paths))
	for i, path := range paths {
		names[i] = strings.Join(path, ".")
	}
	return names
}

func columnsOf(schema *parquet.Schema) []ParquetColumn {
	paths := schema.Columns()
	cols := make([]ParquetColumn, len(paths))
	for i, path := range paths {
		typeName := "unknown"
		if leaf, ok := schema.Lookup(path...); ok {
			typeName = leaf.Node.Type().String()
		}
		cols[i] = ParquetColumn{Name: strings.Join(path, "."), Type: typeName}
	}
	return cols
}

func valueToGo(v parquet.Value) interface{} {
	if v.IsNull() {
		return nil
	}
	switch v.Kind() {
	case parquet.Boolean:
		return v.Boolean()
	case parquet.Int32:
		return int64(v.Int32())
	case parquet.Int64:
		return v.Int64()
	case parquet.Float:
		return float64(v.Float())
	case parquet.Double:
		return v.Double()
	case parquet.ByteArray, parquet.FixedLenByteArray:
		return string(v.Bytes())
	default:
		return v.String()
	}
}

// rowToMap converts a flat parquet.Row to a name-keyed map using each value's own column index.
func rowToMap(row parquet.Row, names []string) map[string]interface{} {
	m := make(map[string]interface{}, len(names))
	for _, v := range row {
		col := v.Column()
		if col < 0 || col >= len(names) {
			continue
		}
		m[names[col]] = valueToGo(v)
	}
	return m
}

// OpenParquetFile opens (or reuses an already-open handle for) path and returns its schema and size.
func (s *ParquetService) OpenParquetFile(path string) (*ParquetFileInfo, error) {
	h, err := s.getOrOpen(path)
	if err != nil {
		return nil, err
	}

	stat, err := h.file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", path, err)
	}

	schema := h.reader.Schema()
	return &ParquetFileInfo{
		Path:         path,
		Columns:      columnsOf(schema),
		NumRows:      h.reader.NumRows(),
		NumRowGroups: h.rowGroups,
		FileSize:     stat.Size(),
	}, nil
}

// ReadParquetPage seeks the cached reader to offset and reads up to limit rows via the row-group offset index, touching only the bytes covering the requested rows.
func (s *ParquetService) ReadParquetPage(path string, offset, limit int) ([]map[string]interface{}, error) {
	h, err := s.getOrOpen(path)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := h.reader.SeekToRow(int64(offset)); err != nil {
		return nil, fmt.Errorf("failed to seek to row %d in %s (check the network drive is still connected): %w", offset, path, err)
	}

	rows := make([]parquet.Row, limit)
	n, err := h.reader.ReadRows(rows)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read rows from %s (check the network drive is still connected): %w", path, err)
	}
	rows = rows[:n]

	names := columnNamesOf(h.reader.Schema())
	result := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		result[i] = rowToMap(row, names)
	}
	return result, nil
}

// ExportParquetToCSV streams the file sequentially through a CSV writer, using its own reader independent of any cached browsing handle for path.
func (s *ParquetService) ExportParquetToCSV(path, outputPath string, columns []string, delimiter rune) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s (check the network drive is still connected): %w", path, err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat %s: %w", path, err)
	}

	pf, err := parquet.OpenFile(f, stat.Size())
	if err != nil {
		return fmt.Errorf("failed to read parquet metadata from %s: %w", path, err)
	}

	reader := parquet.NewGenericReader[any](pf)
	defer reader.Close()

	allNames := columnNamesOf(reader.Schema())
	selected := columns
	if len(selected) == 0 {
		selected = allNames
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

	s.progress.EmitStart(ProgressTypeGeneric, "parquet-export", fmt.Sprintf("Exporting %s...", filepath.Base(path)))

	total := reader.NumRows()
	var written int64
	const batchSize = 1000
	buf := make([]parquet.Row, batchSize)
	record := make([]string, len(selected))

	for {
		n, readErr := reader.ReadRows(buf)
		for i := 0; i < n; i++ {
			m := rowToMap(buf[i], allNames)
			for j, name := range selected {
				record[j] = ""
				if v, ok := m[name]; ok && v != nil {
					record[j] = fmt.Sprintf("%v", v)
				}
			}
			if err := w.Write(record); err != nil {
				s.progress.EmitError(ProgressTypeGeneric, "parquet-export", "Export failed", err.Error())
				return fmt.Errorf("failed to write row: %w", err)
			}
		}
		written += int64(n)
		if total > 0 && n > 0 {
			pct := float64(written) / float64(total) * 100
			s.progress.EmitProgress(ProgressTypeGeneric, "parquet-export", fmt.Sprintf("Exported %d/%d rows", written, total), pct)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			s.progress.EmitError(ProgressTypeGeneric, "parquet-export", "Export failed", readErr.Error())
			return fmt.Errorf("failed to read rows from %s (check the network drive is still connected): %w", path, readErr)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		s.progress.EmitError(ProgressTypeGeneric, "parquet-export", "Export failed", err.Error())
		return fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	s.progress.EmitComplete(ProgressTypeGeneric, "parquet-export", fmt.Sprintf("Exported %d rows to %s", written, filepath.Base(outputPath)))
	return nil
}

// CloseParquetFile releases the cached handle for path, if one is open.
func (s *ParquetService) CloseParquetFile(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	h, ok := s.handles[path]
	if !ok {
		return nil
	}
	delete(s.handles, path)

	readerErr := h.reader.Close()
	fileErr := h.file.Close()
	if readerErr != nil {
		return readerErr
	}
	return fileErr
}
