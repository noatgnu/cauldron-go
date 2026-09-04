package services

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestTableFileService(t *testing.T) *TableFileService {
	t.Helper()
	notifier := NewProgressNotifier(nil)
	return NewTableFileService(NewParquetService(notifier), NewDelimitedFileService(notifier))
}

func TestTableFileService_DispatchesParquetByExtension(t *testing.T) {
	path := writeTestParquetFile(t, makeTestRows(5), 5)
	svc := newTestTableFileService(t)
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	info, err := svc.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if info.NumRows != 5 {
		t.Errorf("expected 5 rows, got %d", info.NumRows)
	}
	if len(svc.parquet.handles) != 1 {
		t.Errorf("expected the parquet service to have cached the handle, got %d", len(svc.parquet.handles))
	}
	if len(svc.delimited.handles) != 0 {
		t.Errorf("expected the delimited service to have no cached handles, got %d", len(svc.delimited.handles))
	}
}

func TestTableFileService_DispatchesCSVByExtension(t *testing.T) {
	path := writeTestDelimitedFile(t, "test.csv", makeCSVContent(5))
	svc := newTestTableFileService(t)
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	info, err := svc.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if info.NumRows != 5 {
		t.Errorf("expected 5 rows, got %d", info.NumRows)
	}
	if len(svc.delimited.handles) != 1 {
		t.Errorf("expected the delimited service to have cached the handle, got %d", len(svc.delimited.handles))
	}
	if len(svc.parquet.handles) != 0 {
		t.Errorf("expected the parquet service to have no cached handles, got %d", len(svc.parquet.handles))
	}
}

func TestTableFileService_DispatchesTSVByExtension(t *testing.T) {
	path := writeTestDelimitedFile(t, "test.tsv", []byte("a\tb\n1\t2\n"))
	svc := newTestTableFileService(t)
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	page, err := svc.ReadPage(path, 0, 1)
	if err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}
	if len(page) != 1 || page[0]["a"] != "1" {
		t.Fatalf("expected tab-delimited row to parse correctly, got %+v", page)
	}
}

func TestTableFileService_CloseOnlyAffectsMatchingService(t *testing.T) {
	parquetPath := writeTestParquetFile(t, makeTestRows(3), 3)
	csvPath := writeTestDelimitedFile(t, "test.csv", makeCSVContent(3))
	svc := newTestTableFileService(t)
	t.Cleanup(func() {
		_ = svc.CloseFile(parquetPath)
		_ = svc.CloseFile(csvPath)
	})

	if _, err := svc.OpenFile(parquetPath); err != nil {
		t.Fatalf("OpenFile(parquet) failed: %v", err)
	}
	if _, err := svc.OpenFile(csvPath); err != nil {
		t.Fatalf("OpenFile(csv) failed: %v", err)
	}

	if err := svc.CloseFile(parquetPath); err != nil {
		t.Fatalf("CloseFile(parquet) failed: %v", err)
	}
	if len(svc.parquet.handles) != 0 {
		t.Errorf("expected parquet handle closed, got %d remaining", len(svc.parquet.handles))
	}
	if len(svc.delimited.handles) != 1 {
		t.Errorf("expected csv handle untouched, got %d", len(svc.delimited.handles))
	}
}

func TestTableFileService_ExportFile_DispatchesByExtension(t *testing.T) {
	path := writeTestDelimitedFile(t, "test.csv", makeCSVContent(2))
	svc := newTestTableFileService(t)
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	outPath := filepath.Join(t.TempDir(), "out.csv")
	if err := svc.ExportFile(path, outPath, nil, ','); err != nil {
		t.Fatalf("ExportFile failed: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected export output file to exist: %v", err)
	}
}
