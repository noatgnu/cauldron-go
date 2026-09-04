package main

import (
	"strings"
	"testing"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid",
			config: &Config{
				InputFile:   "test.csv",
				IDVars:      "id",
				NamesFrom:   "sample",
				ValuesFrom:  "intensity",
				OnDuplicate: "error",
			},
		},
		{
			name:    "missing input",
			config:  &Config{IDVars: "id", NamesFrom: "sample", ValuesFrom: "intensity", OnDuplicate: "error"},
			wantErr: true,
			errMsg:  "--input is required",
		},
		{
			name:    "missing id-vars",
			config:  &Config{InputFile: "test.csv", NamesFrom: "sample", ValuesFrom: "intensity", OnDuplicate: "error"},
			wantErr: true,
			errMsg:  "--id-vars is required",
		},
		{
			name:    "missing names-from",
			config:  &Config{InputFile: "test.csv", IDVars: "id", ValuesFrom: "intensity", OnDuplicate: "error"},
			wantErr: true,
			errMsg:  "--names-from is required",
		},
		{
			name:    "missing values-from",
			config:  &Config{InputFile: "test.csv", IDVars: "id", NamesFrom: "sample", OnDuplicate: "error"},
			wantErr: true,
			errMsg:  "--values-from is required",
		},
		{
			name: "invalid on-duplicate",
			config: &Config{
				InputFile:   "test.csv",
				IDVars:      "id",
				NamesFrom:   "sample",
				ValuesFrom:  "intensity",
				OnDuplicate: "bogus",
			},
			wantErr: true,
			errMsg:  "--on-duplicate must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateConfig() expected error but got none")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateConfig() error = %v, should contain %v", err, tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("validateConfig() unexpected error: %v", err)
			}
		})
	}
}
