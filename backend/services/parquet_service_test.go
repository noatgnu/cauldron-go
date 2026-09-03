package services

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parquet-go/parquet-go"
)

type parquetTestRow struct {
	ID    int64   `parquet:"id"`
	Name  string  `parquet:"name"`
	Score float64 `parquet:"score"`
}

// writeTestParquetFile writes rows in batches of batchSize, flushing between batches to produce multiple row groups.
func writeTestParquetFile(t *testing.T, rows []parquetTestRow, batchSize int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.parquet")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test parquet file: %v", err)
	}
	defer f.Close()

	w := parquet.NewGenericWriter[parquetTestRow](f)
	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}
		if _, err := w.Write(rows[i:end]); err != nil {
			t.Fatalf("failed to write rows: %v", err)
		}
		if end < len(rows) {
			if err := w.Flush(); err != nil {
				t.Fatalf("failed to flush row group: %v", err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close parquet writer: %v", err)
	}

	return path
}

func makeTestRows(n int) []parquetTestRow {
	rows := make([]parquetTestRow, n)
	for i := 0; i < n; i++ {
		rows[i] = parquetTestRow{ID: int64(i), Name: "row-" + string(rune('a'+i%26)), Score: float64(i) * 1.5}
	}
	return rows
}

func TestOpenParquetFile_ReportsSchemaAndRowCount(t *testing.T) {
	path := writeTestParquetFile(t, makeTestRows(10), 5)
	svc := NewParquetService(NewProgressNotifier(nil))

	info, err := svc.OpenParquetFile(path)
	if err != nil {
		t.Fatalf("OpenParquetFile failed: %v", err)
	}

	if info.NumRows != 10 {
		t.Errorf("expected 10 rows, got %d", info.NumRows)
	}
	if info.NumRowGroups != 2 {
		t.Errorf("expected 2 row groups, got %d", info.NumRowGroups)
	}
	if len(info.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d: %+v", len(info.Columns), info.Columns)
	}
	names := map[string]bool{}
	for _, c := range info.Columns {
		names[c.Name] = true
	}
	for _, want := range []string{"id", "name", "score"} {
		if !names[want] {
			t.Errorf("expected column %q in schema, got %+v", want, info.Columns)
		}
	}
}

func TestOpenParquetFile_CachesHandleAcrossCalls(t *testing.T) {
	path := writeTestParquetFile(t, makeTestRows(5), 5)
	svc := NewParquetService(NewProgressNotifier(nil))

	if _, err := svc.OpenParquetFile(path); err != nil {
		t.Fatalf("first OpenParquetFile failed: %v", err)
	}
	if _, err := svc.OpenParquetFile(path); err != nil {
		t.Fatalf("second OpenParquetFile failed: %v", err)
	}

	if len(svc.handles) != 1 {
		t.Errorf("expected exactly 1 cached handle after repeated opens, got %d", len(svc.handles))
	}
}

func TestReadParquetPage_ReturnsCorrectSlice(t *testing.T) {
	path := writeTestParquetFile(t, makeTestRows(10), 5)
	svc := NewParquetService(NewProgressNotifier(nil))

	page, err := svc.ReadParquetPage(path, 3, 4)
	if err != nil {
		t.Fatalf("ReadParquetPage failed: %v", err)
	}
	if len(page) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(page))
	}
	for i, row := range page {
		wantID := int64(3 + i)
		if id, _ := row["id"].(int64); id != wantID {
			t.Errorf("row %d: expected id %d, got %v", i, wantID, row["id"])
		}
	}
}

func TestReadParquetPage_PartialLastPage(t *testing.T) {
	path := writeTestParquetFile(t, makeTestRows(10), 5)
	svc := NewParquetService(NewProgressNotifier(nil))

	page, err := svc.ReadParquetPage(path, 8, 5)
	if err != nil {
		t.Fatalf("ReadParquetPage failed: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected a partial page of 2 rows, got %d", len(page))
	}
	if id, _ := page[0]["id"].(int64); id != 8 {
		t.Errorf("expected first row id 8, got %v", page[0]["id"])
	}
	if id, _ := page[1]["id"].(int64); id != 9 {
		t.Errorf("expected second row id 9, got %v", page[1]["id"])
	}
}

func TestReadParquetPage_SpansTwoRowGroups(t *testing.T) {
	path := writeTestParquetFile(t, makeTestRows(10), 4) // row groups: [0-3],[4-7],[8-9]
	svc := NewParquetService(NewProgressNotifier(nil))

	page, err := svc.ReadParquetPage(path, 2, 6) // spans row group 0 and row group 1
	if err != nil {
		t.Fatalf("ReadParquetPage failed: %v", err)
	}
	if len(page) != 6 {
		t.Fatalf("expected 6 rows, got %d", len(page))
	}
	for i, row := range page {
		wantID := int64(2 + i)
		if id, _ := row["id"].(int64); id != wantID {
			t.Errorf("row %d: expected id %d, got %v", i, wantID, row["id"])
		}
	}
}

func TestExportParquetToCSV_RoundTrips(t *testing.T) {
	path := writeTestParquetFile(t, makeTestRows(7), 3)
	svc := NewParquetService(NewProgressNotifier(nil))

	outPath := filepath.Join(t.TempDir(), "out.csv")
	if err := svc.ExportParquetToCSV(path, outPath, nil, ','); err != nil {
		t.Fatalf("ExportParquetToCSV failed: %v", err)
	}

	records := readCSV(t, outPath)
	if len(records) != 8 { // header + 7 rows
		t.Fatalf("expected 8 CSV records (header + 7 rows), got %d", len(records))
	}
	if records[0][0] != "id" || records[0][1] != "name" || records[0][2] != "score" {
		t.Errorf("unexpected header: %v", records[0])
	}
	if records[1][0] != "0" {
		t.Errorf("expected first data row id '0', got %v", records[1])
	}
}

func TestExportParquetToCSV_ColumnSubset(t *testing.T) {
	path := writeTestParquetFile(t, makeTestRows(3), 3)
	svc := NewParquetService(NewProgressNotifier(nil))

	outPath := filepath.Join(t.TempDir(), "out.csv")
	if err := svc.ExportParquetToCSV(path, outPath, []string{"name", "id"}, ','); err != nil {
		t.Fatalf("ExportParquetToCSV failed: %v", err)
	}

	records := readCSV(t, outPath)
	if len(records) != 4 {
		t.Fatalf("expected 4 CSV records, got %d", len(records))
	}
	if records[0][0] != "name" || records[0][1] != "id" {
		t.Fatalf("expected header [name id], got %v", records[0])
	}
	if len(records[1]) != 2 {
		t.Fatalf("expected 2 columns per row, got %d", len(records[1]))
	}
}

func TestExportParquetToCSV_TabDelimiter(t *testing.T) {
	path := writeTestParquetFile(t, makeTestRows(2), 2)
	svc := NewParquetService(NewProgressNotifier(nil))

	outPath := filepath.Join(t.TempDir(), "out.tsv")
	if err := svc.ExportParquetToCSV(path, outPath, nil, '\t'); err != nil {
		t.Fatalf("ExportParquetToCSV failed: %v", err)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 3 { // header + 2 rows
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), string(raw))
	}
	wantHeader := "id\tname\tscore"
	if lines[0] != wantHeader {
		t.Errorf("expected tab-delimited header %q, got %q", wantHeader, lines[0])
	}
}

func TestCloseParquetFile_RemovesHandleAndAllowsReopen(t *testing.T) {
	path := writeTestParquetFile(t, makeTestRows(5), 5)
	svc := NewParquetService(NewProgressNotifier(nil))

	if _, err := svc.OpenParquetFile(path); err != nil {
		t.Fatalf("OpenParquetFile failed: %v", err)
	}
	if len(svc.handles) != 1 {
		t.Fatalf("expected 1 cached handle, got %d", len(svc.handles))
	}

	if err := svc.CloseParquetFile(path); err != nil {
		t.Fatalf("CloseParquetFile failed: %v", err)
	}
	if len(svc.handles) != 0 {
		t.Errorf("expected 0 cached handles after close, got %d", len(svc.handles))
	}

	if _, err := svc.OpenParquetFile(path); err != nil {
		t.Fatalf("reopening after close failed: %v", err)
	}
	if len(svc.handles) != 1 {
		t.Errorf("expected 1 cached handle after reopen, got %d", len(svc.handles))
	}
}

func TestCloseParquetFile_UnknownPathIsNoOp(t *testing.T) {
	svc := NewParquetService(NewProgressNotifier(nil))
	if err := svc.CloseParquetFile("/does/not/exist.parquet"); err != nil {
		t.Errorf("expected no error closing an unopened path, got %v", err)
	}
}

func TestOpenParquetFile_MissingFileReturnsClearError(t *testing.T) {
	svc := NewParquetService(NewProgressNotifier(nil))
	_, err := svc.OpenParquetFile(filepath.Join(t.TempDir(), "does-not-exist.parquet"))
	if err == nil {
		t.Fatal("expected an error for a missing file, got nil")
	}
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open CSV output: %v", err)
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV output: %v", err)
	}
	return records
}
