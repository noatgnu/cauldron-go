package services

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeTestDelimitedFile(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	return path
}

func makeCSVContent(numRows int) []byte {
	var b strings.Builder
	b.WriteString("id,name,score\n")
	for i := 0; i < numRows; i++ {
		b.WriteString(strconv.Itoa(i))
		b.WriteString(",row")
		b.WriteString(strconv.Itoa(i))
		b.WriteString(",")
		b.WriteString(strconv.Itoa(i * 2))
		b.WriteString("\n")
	}
	return []byte(b.String())
}

func TestOpenFile_ReportsHeaderAndRowCount(t *testing.T) {
	path := writeTestDelimitedFile(t, "test.csv", makeCSVContent(10))
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	info, err := svc.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if info.NumRows != 10 {
		t.Errorf("expected 10 rows, got %d", info.NumRows)
	}
	if len(info.Columns) != 3 {
		t.Fatalf("expected 3 columns, got %d: %+v", len(info.Columns), info.Columns)
	}
	names := []string{info.Columns[0].Name, info.Columns[1].Name, info.Columns[2].Name}
	want := []string{"id", "name", "score"}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("column %d: expected %q, got %q", i, want[i], names[i])
		}
	}
}

func TestReadPage_ReturnsCorrectSlice(t *testing.T) {
	path := writeTestDelimitedFile(t, "test.csv", makeCSVContent(10))
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	page, err := svc.ReadPage(path, 3, 4)
	if err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}
	if len(page) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(page))
	}
	for i, row := range page {
		wantID := strconv.Itoa(3 + i)
		if id, _ := row["id"].(string); id != wantID {
			t.Errorf("row %d: expected id %q, got %v", i, wantID, row["id"])
		}
	}
}

func TestReadPage_PartialLastPage(t *testing.T) {
	path := writeTestDelimitedFile(t, "test.csv", makeCSVContent(10))
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	page, err := svc.ReadPage(path, 8, 5)
	if err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected a partial page of 2 rows, got %d", len(page))
	}
	if id, _ := page[0]["id"].(string); id != "8" {
		t.Errorf("expected first row id 8, got %v", page[0]["id"])
	}
	if id, _ := page[1]["id"].(string); id != "9" {
		t.Errorf("expected second row id 9, got %v", page[1]["id"])
	}
}

func TestReadPage_HandlesQuotedCommaAndEmbeddedNewline(t *testing.T) {
	content := "id,name,note\n" +
		"0,alice,\"hello, world\"\n" +
		"1,\"bob\nnewline\",x\n" +
		"2,carol,y\n"
	path := writeTestDelimitedFile(t, "test.csv", []byte(content))
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	info, err := svc.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if info.NumRows != 3 {
		t.Fatalf("expected 3 rows, got %d", info.NumRows)
	}

	page, err := svc.ReadPage(path, 0, 3)
	if err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}
	if len(page) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(page))
	}
	if page[0]["note"] != "hello, world" {
		t.Errorf("expected quoted-comma field to round-trip, got %v", page[0]["note"])
	}
	if page[1]["name"] != "bob\nnewline" {
		t.Errorf("expected quoted embedded newline to round-trip, got %q", page[1]["name"])
	}

	// Seeking directly to the row after the tricky quoted-newline row must land correctly,
	// proving the byte-offset index treats the embedded newline as part of one row, not two.
	pageAfter, err := svc.ReadPage(path, 2, 1)
	if err != nil {
		t.Fatalf("ReadPage from offset 2 failed: %v", err)
	}
	if len(pageAfter) != 1 || pageAfter[0]["name"] != "carol" {
		t.Fatalf("expected row 2 to be carol, got %+v", pageAfter)
	}
}

func TestOpenFile_TSVExtensionUsesTabDelimiter(t *testing.T) {
	content := "id\tname\n0\talice\n1\tbob\n"
	path := writeTestDelimitedFile(t, "test.tsv", []byte(content))
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	info, err := svc.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if info.NumRows != 2 {
		t.Fatalf("expected 2 rows, got %d", info.NumRows)
	}

	page, err := svc.ReadPage(path, 0, 2)
	if err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}
	if page[0]["name"] != "alice" || page[1]["name"] != "bob" {
		t.Fatalf("expected tab-delimited fields to parse correctly, got %+v", page)
	}
}

func TestOpenFile_RaggedRowsDoNotAbortIndexing(t *testing.T) {
	content := "a,b,c\n1,2,3\n4,5\n6,7,8,9\n"
	path := writeTestDelimitedFile(t, "test.csv", []byte(content))
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	info, err := svc.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if info.NumRows != 3 {
		t.Fatalf("expected 3 rows despite ragged field counts, got %d", info.NumRows)
	}

	page, err := svc.ReadPage(path, 1, 1)
	if err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}
	if page[0]["a"] != "4" || page[0]["b"] != "5" || page[0]["c"] != "" {
		t.Fatalf("expected short row to fill missing trailing field with empty string, got %+v", page[0])
	}
}

func TestOpenFile_EmptyFileReturnsZeroRows(t *testing.T) {
	path := writeTestDelimitedFile(t, "empty.csv", []byte(""))
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	info, err := svc.OpenFile(path)
	if err != nil {
		t.Fatalf("expected no error for empty file, got %v", err)
	}
	if info.NumRows != 0 {
		t.Errorf("expected 0 rows, got %d", info.NumRows)
	}
	if len(info.Columns) != 0 {
		t.Errorf("expected 0 columns, got %d", len(info.Columns))
	}
}

func TestOpenFile_HeaderOnlyFileReturnsZeroRows(t *testing.T) {
	path := writeTestDelimitedFile(t, "headeronly.csv", []byte("a,b,c\n"))
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	info, err := svc.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if info.NumRows != 0 {
		t.Errorf("expected 0 rows, got %d", info.NumRows)
	}
	if len(info.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(info.Columns))
	}
}

func TestOpenFile_StripsLeadingBOM(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	content := append(bom, []byte("a,b,c\n1,2,3\n")...)
	path := writeTestDelimitedFile(t, "bom.csv", content)
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	info, err := svc.OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if info.Columns[0].Name != "a" {
		t.Errorf("expected BOM stripped from first header name, got %q", info.Columns[0].Name)
	}

	page, err := svc.ReadPage(path, 0, 1)
	if err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}
	if page[0]["a"] != "1" {
		t.Errorf("expected first row to parse correctly after BOM, got %+v", page[0])
	}
}

func TestExportToCSV_RoundTrips(t *testing.T) {
	path := writeTestDelimitedFile(t, "test.csv", makeCSVContent(7))
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	outPath := filepath.Join(t.TempDir(), "out.csv")
	if err := svc.ExportToCSV(path, outPath, nil, ','); err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
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

func TestExportToCSV_ColumnSubsetAndDelimiterConversion(t *testing.T) {
	path := writeTestDelimitedFile(t, "test.csv", makeCSVContent(3))
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	outPath := filepath.Join(t.TempDir(), "out.tsv")
	if err := svc.ExportToCSV(path, outPath, []string{"name", "id"}, '\t'); err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 4 { // header + 3 rows
		t.Fatalf("expected 4 lines, got %d: %q", len(lines), string(raw))
	}
	if lines[0] != "name\tid" {
		t.Errorf("expected tab-delimited header 'name\\tid', got %q", lines[0])
	}
}

func TestCloseFile_RemovesHandleAndAllowsReopen(t *testing.T) {
	path := writeTestDelimitedFile(t, "test.csv", makeCSVContent(5))
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	t.Cleanup(func() { _ = svc.CloseFile(path) })

	if _, err := svc.OpenFile(path); err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	if len(svc.handles) != 1 {
		t.Fatalf("expected 1 cached handle, got %d", len(svc.handles))
	}

	if err := svc.CloseFile(path); err != nil {
		t.Fatalf("CloseFile failed: %v", err)
	}
	if len(svc.handles) != 0 {
		t.Errorf("expected 0 cached handles after close, got %d", len(svc.handles))
	}

	if _, err := svc.OpenFile(path); err != nil {
		t.Fatalf("reopening after close failed: %v", err)
	}
	if len(svc.handles) != 1 {
		t.Errorf("expected 1 cached handle after reopen, got %d", len(svc.handles))
	}
}

func TestCloseFile_UnknownPathIsNoOp(t *testing.T) {
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	if err := svc.CloseFile("/does/not/exist.csv"); err != nil {
		t.Errorf("expected no error closing an unopened path, got %v", err)
	}
}

func TestOpenFile_MissingFileReturnsClearError(t *testing.T) {
	svc := NewDelimitedFileService(NewProgressNotifier(nil))
	_, err := svc.OpenFile(filepath.Join(t.TempDir(), "does-not-exist.csv"))
	if err == nil {
		t.Fatal("expected an error for a missing file, got nil")
	}
}
