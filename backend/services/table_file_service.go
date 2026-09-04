package services

import (
	"path/filepath"
	"strings"
)

// TableFileService dispatches tabular file operations to ParquetService or DelimitedFileService based on file extension.
type TableFileService struct {
	parquet   *ParquetService
	delimited *DelimitedFileService
}

func NewTableFileService(parquet *ParquetService, delimited *DelimitedFileService) *TableFileService {
	return &TableFileService{parquet: parquet, delimited: delimited}
}

func isParquet(path string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".parquet"
}

// OpenFile opens path and returns its schema and size.
func (s *TableFileService) OpenFile(path string) (*DataFileInfo, error) {
	if isParquet(path) {
		return s.parquet.OpenParquetFile(path)
	}
	return s.delimited.OpenFile(path)
}

// ReadPage reads up to limit rows starting at offset.
func (s *TableFileService) ReadPage(path string, offset, limit int) ([]map[string]interface{}, error) {
	if isParquet(path) {
		return s.parquet.ReadParquetPage(path, offset, limit)
	}
	return s.delimited.ReadPage(path, offset, limit)
}

// ExportFile exports rows [offset, offset+limit) and the selected columns of path to outputPath, delimited by delimiter. offset<=0 starts from the first row; limit<=0 means no upper bound.
func (s *TableFileService) ExportFile(path, outputPath string, columns []string, delimiter rune, offset, limit int) error {
	if isParquet(path) {
		return s.parquet.ExportParquetToCSV(path, outputPath, columns, delimiter, offset, limit)
	}
	return s.delimited.ExportToCSV(path, outputPath, columns, delimiter, offset, limit)
}

// CloseFile releases the cached handle for path, if one is open.
func (s *TableFileService) CloseFile(path string) error {
	if isParquet(path) {
		return s.parquet.CloseParquetFile(path)
	}
	return s.delimited.CloseFile(path)
}
