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
			name:   "valid",
			config: &Config{InputFile: "test.csv", IDVars: "id"},
		},
		{
			name:    "missing input",
			config:  &Config{IDVars: "id"},
			wantErr: true,
			errMsg:  "--input is required",
		},
		{
			name:    "missing id-vars",
			config:  &Config{InputFile: "test.csv"},
			wantErr: true,
			errMsg:  "--id-vars is required",
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
