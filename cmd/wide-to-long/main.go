package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/noatgnu/cauldron-go/internal/reshape"
)

type Config struct {
	InputFile    string
	OutputFolder string
	IDVars       string
	ValueVars    string
	VarName      string
	ValueName    string
	Delimiter    string
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
	flag.StringVar(&config.IDVars, "id-vars", "", "Comma-separated identifier columns to keep as-is")
	flag.StringVar(&config.ValueVars, "value-vars", "", "Comma-separated columns to melt (default: all columns not in --id-vars)")
	flag.StringVar(&config.VarName, "var-name", "variable", "Name of the new column holding the melted columns' original names")
	flag.StringVar(&config.ValueName, "value-name", "value", "Name of the new column holding the melted cell values")
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
	return nil
}

func run(config *Config) error {
	if err := validateConfig(config); err != nil {
		return err
	}

	if err := os.MkdirAll(config.OutputFolder, 0755); err != nil {
		return fmt.Errorf("failed to create output folder: %w", err)
	}

	fmt.Println("CauldronGO Wide to Long (Melt)")
	fmt.Println("==============================")

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
	valueVars := reshape.SplitList(config.ValueVars)

	outHeader, outRows, err := reshape.Melt(header, rows, idVars, valueVars, config.VarName, config.ValueName)
	if err != nil {
		return fmt.Errorf("failed to melt table: %w", err)
	}

	outputPath := filepath.Join(config.OutputFolder, "melted.data.tsv")
	if err := reshape.WriteTable(outputPath, delim, outHeader, outRows); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("\n[SUCCESS] Results written to: %s\n", outputPath)
	fmt.Printf("Output rows: %d\n", len(outRows))
	return nil
}
