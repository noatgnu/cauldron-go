// Package reshape provides table reshaping primitives (melt/pivot) shared by
// the wide-to-long and long-to-wide native plugin binaries.
package reshape

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

// DuplicateStrategy controls how Pivot resolves multiple input rows that map
// to the same id-vars + names-from key.
type DuplicateStrategy string

const (
	DuplicateError  DuplicateStrategy = "error"
	DuplicateFirst  DuplicateStrategy = "first"
	DuplicateLast   DuplicateStrategy = "last"
	DuplicateConcat DuplicateStrategy = "concat"
)

// PivotOptions configures Pivot's behavior beyond the required id/names/values columns.
type PivotOptions struct {
	NamesPrefix     string
	FillValue       string
	OnDuplicate     DuplicateStrategy
	ConcatSeparator string
}

// ReadTable reads a delimited file into a header row and the remaining data rows.
func ReadTable(path string, delim rune) (header []string, rows [][]string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = delim
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	header, err = reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read header: %w", err)
	}

	rows, err = reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read rows: %w", err)
	}

	return header, rows, nil
}

// WriteTable writes a header row and data rows to a delimited file.
func WriteTable(path string, delim rune, header []string, rows [][]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = delim

	records := make([][]string, 0, len(rows)+1)
	records = append(records, header)
	records = append(records, rows...)

	if err := writer.WriteAll(records); err != nil {
		return err
	}
	return writer.Error()
}

// ParseDelimiter validates that s is a single-character delimiter and returns it as a rune.
func ParseDelimiter(s string) (rune, error) {
	runes := []rune(s)
	if len(runes) != 1 {
		return 0, fmt.Errorf("delimiter must be a single character, got %q", s)
	}
	return runes[0], nil
}

// SplitList splits a comma-separated flag value into trimmed, non-empty parts.
// An empty or whitespace-only input returns nil.
func SplitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ResolveColumns maps names to their positions in header, in the order given.
// It errors on a duplicate column name in header, a name requested more than
// once, or a name not present in header.
func ResolveColumns(header []string, names []string) ([]int, error) {
	index := make(map[string]int, len(header))
	for i, h := range header {
		trimmed := strings.TrimSpace(h)
		if _, exists := index[trimmed]; exists {
			return nil, fmt.Errorf("duplicate column name %q in header", trimmed)
		}
		index[trimmed] = i
	}

	seen := make(map[string]bool, len(names))
	indices := make([]int, 0, len(names))
	for _, name := range names {
		if seen[name] {
			return nil, fmt.Errorf("column %q requested more than once", name)
		}
		seen[name] = true

		i, ok := index[name]
		if !ok {
			return nil, fmt.Errorf("column %q not found. Available: %s", name, strings.Join(header, ", "))
		}
		indices = append(indices, i)
	}
	return indices, nil
}

// ResolveColumn resolves a single column name to its header index.
func ResolveColumn(header []string, name string) (int, error) {
	indices, err := ResolveColumns(header, []string{name})
	if err != nil {
		return -1, err
	}
	return indices[0], nil
}

// cellAt returns row[i], or "" if the row is short a trailing cell.
func cellAt(row []string, i int) string {
	if i < len(row) {
		return row[i]
	}
	return ""
}

// Melt collapses valueVars columns into variable/value pairs, keeping idVars
// as-is on every output row. If valueVars is empty, every column not in
// idVars is melted (matching pandas melt / tidyr pivot_longer defaults).
func Melt(header []string, rows [][]string, idVars, valueVars []string, varName, valueName string) ([]string, [][]string, error) {
	if len(idVars) == 0 {
		return nil, nil, fmt.Errorf("at least one id-var column is required")
	}

	idIndices, err := ResolveColumns(header, idVars)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving id-vars: %w", err)
	}

	idSet := make(map[int]bool, len(idIndices))
	for _, i := range idIndices {
		idSet[i] = true
	}

	var valueIndices []int
	if len(valueVars) == 0 {
		for i := range header {
			if !idSet[i] {
				valueIndices = append(valueIndices, i)
			}
		}
	} else {
		valueIndices, err = ResolveColumns(header, valueVars)
		if err != nil {
			return nil, nil, fmt.Errorf("resolving value-vars: %w", err)
		}
		for _, i := range valueIndices {
			if idSet[i] {
				return nil, nil, fmt.Errorf("column %q cannot be both an id-var and a value-var", header[i])
			}
		}
	}

	if len(valueIndices) == 0 {
		return nil, nil, fmt.Errorf("no value columns to melt (all columns are id-vars)")
	}

	outHeader := make([]string, 0, len(idIndices)+2)
	for _, i := range idIndices {
		outHeader = append(outHeader, header[i])
	}
	outHeader = append(outHeader, varName, valueName)

	outRows := make([][]string, 0, len(rows)*len(valueIndices))
	for _, row := range rows {
		for _, vi := range valueIndices {
			outRow := make([]string, 0, len(idIndices)+2)
			for _, ii := range idIndices {
				outRow = append(outRow, cellAt(row, ii))
			}
			outRow = append(outRow, header[vi], cellAt(row, vi))
			outRows = append(outRows, outRow)
		}
	}

	return outHeader, outRows, nil
}

// Pivot spreads namesFrom/valuesFrom back into separate columns, one output
// row per distinct idVars combination. New column order follows the
// first-appearance order of namesFrom values (matching tidyr pivot_wider).
func Pivot(header []string, rows [][]string, idVars []string, namesFrom, valuesFrom string, opts PivotOptions) ([]string, [][]string, error) {
	if len(idVars) == 0 {
		return nil, nil, fmt.Errorf("at least one id-var column is required")
	}

	idIndices, err := ResolveColumns(header, idVars)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving id-vars: %w", err)
	}

	namesFromIndex, err := ResolveColumn(header, namesFrom)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving names-from: %w", err)
	}
	valuesFromIndex, err := ResolveColumn(header, valuesFrom)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving values-from: %w", err)
	}
	if namesFromIndex == valuesFromIndex {
		return nil, nil, fmt.Errorf("names-from and values-from must be different columns")
	}

	idSet := make(map[int]bool, len(idIndices))
	for _, i := range idIndices {
		idSet[i] = true
	}
	if idSet[namesFromIndex] {
		return nil, nil, fmt.Errorf("names-from column %q cannot also be an id-var", namesFrom)
	}
	if idSet[valuesFromIndex] {
		return nil, nil, fmt.Errorf("values-from column %q cannot also be an id-var", valuesFrom)
	}

	if opts.OnDuplicate == "" {
		opts.OnDuplicate = DuplicateError
	}
	if opts.ConcatSeparator == "" {
		opts.ConcatSeparator = ";"
	}

	var nameOrder []string
	nameSeen := make(map[string]bool)

	var groupOrder []string
	groupSeen := make(map[string]bool)
	groupIDValues := make(map[string][]string)
	cells := make(map[string]map[string]string)
	firstRowNum := make(map[string]map[string]int)

	for rowNum, row := range rows {
		idKeyParts := make([]string, 0, len(idIndices))
		for _, ii := range idIndices {
			idKeyParts = append(idKeyParts, cellAt(row, ii))
		}
		key := strings.Join(idKeyParts, "\x1f")

		if !groupSeen[key] {
			groupSeen[key] = true
			groupOrder = append(groupOrder, key)
			groupIDValues[key] = idKeyParts
			cells[key] = make(map[string]string)
			firstRowNum[key] = make(map[string]int)
		}

		name := cellAt(row, namesFromIndex)
		if !nameSeen[name] {
			nameSeen[name] = true
			nameOrder = append(nameOrder, name)
		}

		value := cellAt(row, valuesFromIndex)
		dataRow := rowNum + 2 // +1 for 0-index, +1 for the header row

		existing, exists := cells[key][name]
		if !exists {
			cells[key][name] = value
			firstRowNum[key][name] = dataRow
			continue
		}

		switch opts.OnDuplicate {
		case DuplicateFirst:
			// keep existing value
		case DuplicateLast:
			cells[key][name] = value
		case DuplicateConcat:
			cells[key][name] = existing + opts.ConcatSeparator + value
		default:
			return nil, nil, fmt.Errorf(
				"duplicate values for id=[%s], %s=%q: row %d and row %d both provide a value (%q vs %q); set --on-duplicate to first, last, or concat to resolve automatically",
				strings.Join(idKeyParts, ", "), namesFrom, name, firstRowNum[key][name], dataRow, existing, value,
			)
		}
	}

	outHeader := make([]string, 0, len(idIndices)+len(nameOrder))
	for _, i := range idIndices {
		outHeader = append(outHeader, header[i])
	}
	for _, name := range nameOrder {
		outHeader = append(outHeader, opts.NamesPrefix+name)
	}

	outRows := make([][]string, 0, len(groupOrder))
	for _, key := range groupOrder {
		outRow := make([]string, 0, len(outHeader))
		outRow = append(outRow, groupIDValues[key]...)
		for _, name := range nameOrder {
			if v, ok := cells[key][name]; ok {
				outRow = append(outRow, v)
			} else {
				outRow = append(outRow, opts.FillValue)
			}
		}
		outRows = append(outRows, outRow)
	}

	return outHeader, outRows, nil
}
