package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/noatgnu/uniprotparser-go"
)

type Config struct {
	InputFile    string
	InputText    string
	OutputFolder string
	Column       string
	From         string
	Fields       string
	Format       string
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
	flag.StringVar(&config.InputFile, "input", "", "Input file path (for file mode)")
	flag.StringVar(&config.InputText, "ids", "", "Comma or newline-separated list of IDs (for text mode)")
	flag.StringVar(&config.OutputFolder, "output", ".", "Output folder")
	flag.StringVar(&config.Column, "column", "", "Column name with protein IDs (required for file mode)")
	flag.StringVar(&config.From, "from", "UniProtKB_AC-ID", "Source database")
	flag.StringVar(&config.Fields, "fields", "accession,id,gene_names,organism_name,protein_name", "Comma-separated fields")
	flag.StringVar(&config.Format, "format", "tsv", "Output format (tsv/json)")
	flag.StringVar(&config.Delimiter, "delimiter", "\t", "Input file delimiter")
	flag.Parse()
	return config
}

func run(config *Config) error {
	if config.InputFile == "" && config.InputText == "" {
		return fmt.Errorf("either --input (file) or --ids (text list) is required")
	}
	if config.InputFile != "" && config.InputText != "" {
		return fmt.Errorf("cannot use both --input and --ids, choose one")
	}
	if config.InputFile != "" && config.Column == "" {
		return fmt.Errorf("--column is required when using --input (file mode)")
	}

	if err := os.MkdirAll(config.OutputFolder, 0755); err != nil {
		return fmt.Errorf("failed to create output folder: %w", err)
	}

	fmt.Println("CauldronGO UniProt Fetcher")
	fmt.Println("==========================")

	var ids []string
	var err error

	if config.InputFile != "" {
		ids, err = readIDsFromFile(config.InputFile, config.Column, config.Delimiter)
		if err != nil {
			return fmt.Errorf("failed to read IDs from file: %w", err)
		}
		if len(ids) == 0 {
			return fmt.Errorf("no IDs found in column '%s'", config.Column)
		}
	} else {
		ids = parseIDsFromText(config.InputText)
		if len(ids) == 0 {
			return fmt.Errorf("no IDs found in text input")
		}
	}

	fmt.Printf("Found %d unique IDs\n", len(ids))

	parser := uniprotparser.NewParser(
		1*time.Second,
		config.Fields,
		config.Format,
		true,
		config.From,
	)

	outputPath := filepath.Join(config.OutputFolder, "uniprot_results.tsv")
	recordCount, err := fetchUniProtData(parser, ids, outputPath, config.Format)
	if err != nil {
		return fmt.Errorf("failed to fetch data: %w", err)
	}

	fmt.Printf("\n[SUCCESS] Results written to: %s\n", outputPath)
	fmt.Printf("Total records: %d\n", recordCount)
	return nil
}

func parseIDsFromText(text string) []string {
	idSet := make(map[string]bool)
	var ids []string

	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';' || r == ' '
	})

	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id != "" && !idSet[id] {
			idSet[id] = true
			ids = append(ids, id)
		}
	}

	return ids
}

func readIDsFromFile(filePath, columnName, delimiter string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = rune(delimiter[0])
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read headers: %w", err)
	}

	columnIndex := -1
	for i, header := range headers {
		if strings.TrimSpace(header) == columnName {
			columnIndex = i
			break
		}
	}

	if columnIndex == -1 {
		return nil, fmt.Errorf("column '%s' not found. Available: %s", columnName, strings.Join(headers, ", "))
	}

	idSet := make(map[string]bool)
	var ids []string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if columnIndex >= len(record) {
			continue
		}

		id := strings.TrimSpace(record[columnIndex])
		if id != "" && !idSet[id] {
			idSet[id] = true
			ids = append(ids, id)
		}
	}

	return ids, nil
}

func fetchUniProtData(parser *uniprotparser.Parser, ids []string, outputPath, format string) (int, error) {
	file, err := os.Create(outputPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	fmt.Printf("\nFetching data for %d IDs (batch size: 500)...\n", len(ids))

	resultChan := parser.Parse(ids, 500)

	recordCount := 0
	headerWritten := false

	for result := range resultChan {
		if result.Data == "" {
			continue
		}

		if format == "tsv" {
			lines := strings.Split(strings.TrimSpace(result.Data), "\n")
			for i, line := range lines {
				if i == 0 && !headerWritten {
					file.WriteString(line + "\n")
					headerWritten = true
				} else if i > 0 || headerWritten {
					if line != "" {
						file.WriteString(line + "\n")
						recordCount++
					}
				}
			}
			fmt.Printf("Batch: %d records\n", len(lines)-1)
		} else {
			file.WriteString(result.Data)
		}
	}

	return recordCount, nil
}
