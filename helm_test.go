package main

import (
	"testing"
)

func TestExtractPackagePath(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		wantPath string
		wantErr  bool
	}{
		{
			name:     "standard output",
			output:   "Successfully packaged chart and saved it to: /tmp/my-chart-1.0.0.tgz\n",
			wantPath: "/tmp/my-chart-1.0.0.tgz",
			wantErr:  false,
		},
		{
			name:     "output with multiple lines",
			output:   "Updating dependencies...\nSaving chart to...\nSuccessfully packaged chart and saved it to: /home/user/charts/app-2.0.0.tgz\n",
			wantPath: "/home/user/charts/app-2.0.0.tgz",
			wantErr:  false,
		},
		{
			name:     "path with spaces",
			output:   "Successfully packaged chart and saved it to: /path/with spaces/chart-1.0.0.tgz\n",
			wantPath: "/path/with spaces/chart-1.0.0.tgz",
			wantErr:  false,
		},
		{
			name:    "no package path in output",
			output:  "Error: chart not found\n",
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := extractPackagePath(tt.output)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if path != tt.wantPath {
				t.Errorf("expected path '%s', got '%s'", tt.wantPath, path)
			}
		})
	}
}

func TestNewHelmCLI(t *testing.T) {
	cli := NewHelmCLI("/path/to/chart")

	if cli.chartPath != "/path/to/chart" {
		t.Errorf("expected chartPath '/path/to/chart', got '%s'", cli.chartPath)
	}
}
