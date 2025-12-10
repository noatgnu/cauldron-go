package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseIDsFromText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "comma separated",
			input:    "P12345,P04637,Q9Y6K9",
			expected: []string{"P12345", "P04637", "Q9Y6K9"},
		},
		{
			name:     "newline separated",
			input:    "P12345\nP04637\nQ9Y6K9",
			expected: []string{"P12345", "P04637", "Q9Y6K9"},
		},
		{
			name:     "space separated",
			input:    "P12345 P04637 Q9Y6K9",
			expected: []string{"P12345", "P04637", "Q9Y6K9"},
		},
		{
			name:     "mixed separators",
			input:    "P12345, P04637\nQ9Y6K9; O15111",
			expected: []string{"P12345", "P04637", "Q9Y6K9", "O15111"},
		},
		{
			name:     "with duplicates",
			input:    "P12345,P12345,P04637",
			expected: []string{"P12345", "P04637"},
		},
		{
			name:     "with whitespace",
			input:    "  P12345  ,  P04637  \n  Q9Y6K9  ",
			expected: []string{"P12345", "P04637", "Q9Y6K9"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "windows line endings",
			input:    "P12345\r\nP04637\r\nQ9Y6K9",
			expected: []string{"P12345", "P04637", "Q9Y6K9"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIDsFromText(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("parseIDsFromText() got %d IDs, want %d", len(result), len(tt.expected))
				t.Errorf("Got: %v", result)
				t.Errorf("Want: %v", tt.expected)
				return
			}

			for i, id := range result {
				if id != tt.expected[i] {
					t.Errorf("parseIDsFromText()[%d] = %v, want %v", i, id, tt.expected[i])
				}
			}
		})
	}
}

func TestReadIDsFromFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		content   string
		column    string
		delimiter string
		expected  []string
		wantErr   bool
	}{
		{
			name:      "tab delimited",
			content:   "ID\tName\nP12345\tProtein1\nP04637\tProtein2\n",
			column:    "ID",
			delimiter: "\t",
			expected:  []string{"P12345", "P04637"},
			wantErr:   false,
		},
		{
			name:      "comma delimited",
			content:   "ID,Name\nP12345,Protein1\nP04637,Protein2\n",
			column:    "ID",
			delimiter: ",",
			expected:  []string{"P12345", "P04637"},
			wantErr:   false,
		},
		{
			name:      "with duplicates",
			content:   "ID\tName\nP12345\tProtein1\nP12345\tProtein1\nP04637\tProtein2\n",
			column:    "ID",
			delimiter: "\t",
			expected:  []string{"P12345", "P04637"},
			wantErr:   false,
		},
		{
			name:      "column not found",
			content:   "ID\tName\nP12345\tProtein1\n",
			column:    "NotFound",
			delimiter: "\t",
			expected:  nil,
			wantErr:   true,
		},
		{
			name:      "empty values filtered",
			content:   "ID\tName\nP12345\tProtein1\n\tProtein2\nP04637\tProtein3\n",
			column:    "ID",
			delimiter: "\t",
			expected:  []string{"P12345", "P04637"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tt.name+".tsv")
			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result, err := readIDsFromFile(testFile, tt.column, tt.delimiter)

			if tt.wantErr {
				if err == nil {
					t.Errorf("readIDsFromFile() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("readIDsFromFile() unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("readIDsFromFile() got %d IDs, want %d", len(result), len(tt.expected))
				return
			}

			for i, id := range result {
				if id != tt.expected[i] {
					t.Errorf("readIDsFromFile()[%d] = %v, want %v", i, id, tt.expected[i])
				}
			}
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "file mode valid",
			config: &Config{
				InputFile:    "test.csv",
				Column:       "ID",
				From:         "UniProtKB_AC-ID",
				Fields:       "accession,id",
				Format:       "tsv",
				OutputFolder: "/tmp",
			},
			wantErr: false,
		},
		{
			name: "text mode valid",
			config: &Config{
				InputText:    "P12345,P04637",
				From:         "UniProtKB_AC-ID",
				Fields:       "accession,id",
				Format:       "tsv",
				OutputFolder: "/tmp",
			},
			wantErr: false,
		},
		{
			name: "no input",
			config: &Config{
				From:         "UniProtKB_AC-ID",
				Fields:       "accession,id",
				Format:       "tsv",
				OutputFolder: "/tmp",
			},
			wantErr: true,
			errMsg:  "either --input (file) or --ids (text list) is required",
		},
		{
			name: "both inputs",
			config: &Config{
				InputFile:    "test.csv",
				InputText:    "P12345",
				Column:       "ID",
				From:         "UniProtKB_AC-ID",
				Fields:       "accession,id",
				Format:       "tsv",
				OutputFolder: "/tmp",
			},
			wantErr: true,
			errMsg:  "cannot use both --input and --ids",
		},
		{
			name: "file mode missing column",
			config: &Config{
				InputFile:    "test.csv",
				From:         "UniProtKB_AC-ID",
				Fields:       "accession,id",
				Format:       "tsv",
				OutputFolder: "/tmp",
			},
			wantErr: true,
			errMsg:  "--column is required when using --input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateConfig() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateConfig() error = %v, should contain %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

func validateConfig(config *Config) error {
	if config.InputFile == "" && config.InputText == "" {
		return &configError{"either --input (file) or --ids (text list) is required"}
	}
	if config.InputFile != "" && config.InputText != "" {
		return &configError{"cannot use both --input and --ids, choose one"}
	}
	if config.InputFile != "" && config.Column == "" {
		return &configError{"--column is required when using --input (file mode)"}
	}
	return nil
}

type configError struct {
	msg string
}

func (e *configError) Error() string {
	return e.msg
}
