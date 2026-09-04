package reshape

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolveColumns(t *testing.T) {
	header := []string{"id", "gene", "sample1", "sample2"}

	t.Run("resolves in requested order", func(t *testing.T) {
		got, err := ResolveColumns(header, []string{"sample2", "id"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []int{3, 0}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("errors on unknown column", func(t *testing.T) {
		_, err := ResolveColumns(header, []string{"missing"})
		if err == nil || !strings.Contains(err.Error(), `column "missing" not found`) {
			t.Fatalf("expected not-found error, got %v", err)
		}
	})

	t.Run("errors on duplicate request", func(t *testing.T) {
		_, err := ResolveColumns(header, []string{"id", "id"})
		if err == nil || !strings.Contains(err.Error(), "requested more than once") {
			t.Fatalf("expected duplicate-request error, got %v", err)
		}
	})

	t.Run("errors on duplicate header name", func(t *testing.T) {
		_, err := ResolveColumns([]string{"id", "id"}, []string{"id"})
		if err == nil || !strings.Contains(err.Error(), "duplicate column name") {
			t.Fatalf("expected duplicate-header error, got %v", err)
		}
	})
}

func TestSplitList(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", nil},
		{"whitespace only", "   ", nil},
		{"single", "id", []string{"id"}},
		{"trims and drops empties", " id , gene ,, sample1", []string{"id", "gene", "sample1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitList(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitList(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDelimiter(t *testing.T) {
	if got, err := ParseDelimiter("\t"); err != nil || got != '\t' {
		t.Errorf("ParseDelimiter(tab) = %q, %v", got, err)
	}
	if got, err := ParseDelimiter(","); err != nil || got != ',' {
		t.Errorf("ParseDelimiter(comma) = %q, %v", got, err)
	}
	if _, err := ParseDelimiter(""); err == nil {
		t.Error("expected error for empty delimiter")
	}
	if _, err := ParseDelimiter("::"); err == nil {
		t.Error("expected error for multi-character delimiter")
	}
}

func TestMelt(t *testing.T) {
	header := []string{"id", "gene", "sample1", "sample2"}
	rows := [][]string{
		{"P1", "GENE1", "10", "20"},
		{"P2", "GENE2", "30", "40"},
	}

	t.Run("explicit value-vars", func(t *testing.T) {
		outHeader, outRows, err := Melt(header, rows, []string{"id", "gene"}, []string{"sample1", "sample2"}, "variable", "value")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantHeader := []string{"id", "gene", "variable", "value"}
		if !reflect.DeepEqual(outHeader, wantHeader) {
			t.Errorf("header = %v, want %v", outHeader, wantHeader)
		}
		wantRows := [][]string{
			{"P1", "GENE1", "sample1", "10"},
			{"P1", "GENE1", "sample2", "20"},
			{"P2", "GENE2", "sample1", "30"},
			{"P2", "GENE2", "sample2", "40"},
		}
		if !reflect.DeepEqual(outRows, wantRows) {
			t.Errorf("rows = %v, want %v", outRows, wantRows)
		}
	})

	t.Run("default value-vars is all non-id columns", func(t *testing.T) {
		outHeader, outRows, err := Melt(header, rows, []string{"id"}, nil, "variable", "value")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantHeader := []string{"id", "variable", "value"}
		if !reflect.DeepEqual(outHeader, wantHeader) {
			t.Errorf("header = %v, want %v", outHeader, wantHeader)
		}
		if len(outRows) != len(rows)*3 { // gene, sample1, sample2
			t.Errorf("got %d output rows, want %d", len(outRows), len(rows)*3)
		}
	})

	t.Run("errors on empty id-vars", func(t *testing.T) {
		_, _, err := Melt(header, rows, nil, []string{"sample1"}, "variable", "value")
		if err == nil || !strings.Contains(err.Error(), "at least one id-var") {
			t.Fatalf("expected empty id-vars error, got %v", err)
		}
	})

	t.Run("errors when id-vars and value-vars overlap", func(t *testing.T) {
		_, _, err := Melt(header, rows, []string{"id"}, []string{"id", "sample1"}, "variable", "value")
		if err == nil || !strings.Contains(err.Error(), "cannot be both an id-var and a value-var") {
			t.Fatalf("expected overlap error, got %v", err)
		}
	})

	t.Run("errors on unknown column", func(t *testing.T) {
		_, _, err := Melt(header, rows, []string{"missing"}, nil, "variable", "value")
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Fatalf("expected not-found error, got %v", err)
		}
	})
}

func TestPivot(t *testing.T) {
	header := []string{"id", "sample", "intensity"}

	t.Run("basic pivot", func(t *testing.T) {
		rows := [][]string{
			{"P1", "s1", "10"},
			{"P1", "s2", "20"},
			{"P2", "s1", "30"},
			{"P2", "s2", "40"},
		}
		outHeader, outRows, err := Pivot(header, rows, []string{"id"}, "sample", "intensity", PivotOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantHeader := []string{"id", "s1", "s2"}
		if !reflect.DeepEqual(outHeader, wantHeader) {
			t.Errorf("header = %v, want %v", outHeader, wantHeader)
		}
		wantRows := [][]string{
			{"P1", "10", "20"},
			{"P2", "30", "40"},
		}
		if !reflect.DeepEqual(outRows, wantRows) {
			t.Errorf("rows = %v, want %v", outRows, wantRows)
		}
	})

	t.Run("fill-value applied to missing combos", func(t *testing.T) {
		rows := [][]string{
			{"P1", "s1", "10"},
			{"P2", "s2", "40"},
		}
		_, outRows, err := Pivot(header, rows, []string{"id"}, "sample", "intensity", PivotOptions{FillValue: "NA"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantRows := [][]string{
			{"P1", "10", "NA"},
			{"P2", "NA", "40"},
		}
		if !reflect.DeepEqual(outRows, wantRows) {
			t.Errorf("rows = %v, want %v", outRows, wantRows)
		}
	})

	t.Run("names-prefix applied", func(t *testing.T) {
		rows := [][]string{{"P1", "s1", "10"}}
		outHeader, _, err := Pivot(header, rows, []string{"id"}, "sample", "intensity", PivotOptions{NamesPrefix: "col_"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"id", "col_s1"}
		if !reflect.DeepEqual(outHeader, want) {
			t.Errorf("header = %v, want %v", outHeader, want)
		}
	})

	t.Run("column order is first-appearance", func(t *testing.T) {
		rows := [][]string{
			{"P1", "s2", "20"},
			{"P1", "s1", "10"},
		}
		outHeader, _, err := Pivot(header, rows, []string{"id"}, "sample", "intensity", PivotOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := []string{"id", "s2", "s1"}
		if !reflect.DeepEqual(outHeader, want) {
			t.Errorf("header = %v, want %v", outHeader, want)
		}
	})

	t.Run("duplicate key errors by default", func(t *testing.T) {
		rows := [][]string{
			{"P1", "s1", "10"},
			{"P1", "s1", "99"},
		}
		_, _, err := Pivot(header, rows, []string{"id"}, "sample", "intensity", PivotOptions{})
		if err == nil {
			t.Fatal("expected duplicate-key error")
		}
		if !strings.Contains(err.Error(), "duplicate values") || !strings.Contains(err.Error(), "row 2") || !strings.Contains(err.Error(), "row 3") {
			t.Errorf("error message missing expected detail: %v", err)
		}
	})

	t.Run("duplicate key first strategy keeps first value", func(t *testing.T) {
		rows := [][]string{
			{"P1", "s1", "10"},
			{"P1", "s1", "99"},
		}
		_, outRows, err := Pivot(header, rows, []string{"id"}, "sample", "intensity", PivotOptions{OnDuplicate: DuplicateFirst})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if outRows[0][1] != "10" {
			t.Errorf("got %q, want first value 10", outRows[0][1])
		}
	})

	t.Run("duplicate key last strategy keeps last value", func(t *testing.T) {
		rows := [][]string{
			{"P1", "s1", "10"},
			{"P1", "s1", "99"},
		}
		_, outRows, err := Pivot(header, rows, []string{"id"}, "sample", "intensity", PivotOptions{OnDuplicate: DuplicateLast})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if outRows[0][1] != "99" {
			t.Errorf("got %q, want last value 99", outRows[0][1])
		}
	})

	t.Run("duplicate key concat strategy joins values", func(t *testing.T) {
		rows := [][]string{
			{"P1", "s1", "10"},
			{"P1", "s1", "99"},
		}
		_, outRows, err := Pivot(header, rows, []string{"id"}, "sample", "intensity", PivotOptions{OnDuplicate: DuplicateConcat, ConcatSeparator: "|"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if outRows[0][1] != "10|99" {
			t.Errorf("got %q, want concatenated 10|99", outRows[0][1])
		}
	})

	t.Run("errors on empty id-vars", func(t *testing.T) {
		_, _, err := Pivot(header, [][]string{{"P1", "s1", "10"}}, nil, "sample", "intensity", PivotOptions{})
		if err == nil || !strings.Contains(err.Error(), "at least one id-var") {
			t.Fatalf("expected empty id-vars error, got %v", err)
		}
	})

	t.Run("errors when names-from equals values-from", func(t *testing.T) {
		_, _, err := Pivot(header, [][]string{{"P1", "s1", "10"}}, []string{"id"}, "sample", "sample", PivotOptions{})
		if err == nil || !strings.Contains(err.Error(), "must be different columns") {
			t.Fatalf("expected same-column error, got %v", err)
		}
	})
}
