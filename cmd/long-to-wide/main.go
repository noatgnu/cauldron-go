package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/noatgnu/cauldron-go/internal/reshape"
)

type Config struct {
	InputFile       string
	OutputFolder    string
	IDVars          string
	NamesFrom       string
	ValuesFrom      string
	NamesPrefix     string
	FillValue       string
	OnDuplicate     string
	ConcatSeparator string
	Delimiter       string
}

func main() {
	config := parseFlags()

	if err := run(config); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() *Config {
	config := &Config{}
	flag.StringVar(&config.InputFile, "input", "", "Input file path")
	flag.StringVar(&config.OutputFolder, "output", ".", "Output folder")
	flag.StringVar(&config.IDVars, "id-vars", "", "Comma-separated columns that uniquely identify an output row")
	flag.StringVar(&config.NamesFrom, "names-from", "", "Column whose distinct values become new column headers")
	flag.StringVar(&config.ValuesFrom, "values-from", "", "Column supplying the cell values")
	flag.StringVar(&config.NamesPrefix, "names-prefix", "", "Prefix prepended to generated column names")
	flag.StringVar(&config.FillValue, "fill-value", "", "Value used for id x names-from combinations absent from the input")
	flag.StringVar(&config.OnDuplicate, "on-duplicate", "error", "How to handle duplicate id+names-from keys: error, first, last, concat")
	flag.StringVar(&config.ConcatSeparator, "concat-separator", ";", "Separator used when --on-duplicate=concat")
	flag.StringVar(&config.Delimiter, "delimiter", "\t", "Input/output file delimiter")
	flag.Parse()
	return config
}

func validateConfig(config *Config) error {
	if config.InputFile == "" {
		return fmt.Errorf("--input is required")
	}
	if config.IDVars == "" {
		return fmt.Errorf("--id-vars is required (at least one identifier column)")
	}
	if config.NamesFrom == "" {
		return fmt.Errorf("--names-from is required")
	}
	if config.ValuesFrom == "" {
		return fmt.Errorf("--values-from is required")
	}
	switch reshape.DuplicateStrategy(config.OnDuplicate) {
	case reshape.DuplicateError, reshape.DuplicateFirst, reshape.DuplicateLast, reshape.DuplicateConcat:
	default:
		return fmt.Errorf("--on-duplicate must be one of error, first, last, concat, got %q", config.OnDuplicate)
	}
	return nil
}

func run(config *Config) error {
	if err := validateConfig(config); err != nil {
		return err
	}

	if err := os.MkdirAll(config.OutputFolder, 0755); err != nil {
		return fmt.Errorf("failed to create output folder: %w", err)
	}

	fmt.Println("CauldronGO Long to Wide (Pivot)")
	fmt.Println("================================")

	delim, err := reshape.ParseDelimiter(config.Delimiter)
	if err != nil {
		return err
	}

	header, rows, err := reshape.ReadTable(config.InputFile, delim)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}
	fmt.Printf("Read %d rows, %d columns\n", len(rows), len(header))

	idVars := reshape.SplitList(config.IDVars)

	opts := reshape.PivotOptions{
		NamesPrefix:     config.NamesPrefix,
		FillValue:       config.FillValue,
		OnDuplicate:     reshape.DuplicateStrategy(config.OnDuplicate),
		ConcatSeparator: config.ConcatSeparator,
	}

	outHeader, outRows, err := reshape.Pivot(header, rows, idVars, config.NamesFrom, config.ValuesFrom, opts)
	if err != nil {
		return fmt.Errorf("failed to pivot table: %w", err)
	}

	outputPath := filepath.Join(config.OutputFolder, "pivoted.data.tsv")
	if err := reshape.WriteTable(outputPath, delim, outHeader, outRows); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("\n[SUCCESS] Results written to: %s\n", outputPath)
	fmt.Printf("Output rows: %d, output columns: %d\n", len(outRows), len(outHeader))
	return nil
}
