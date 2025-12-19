package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseChart(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		wantName    string
		wantVersion string
	}{
		{
			name: "valid chart v2",
			content: `apiVersion: v2
name: my-chart
version: 1.0.0
appVersion: "1.0.0"
description: A sample Helm chart
`,
			wantErr:     false,
			wantName:    "my-chart",
			wantVersion: "1.0.0",
		},
		{
			name: "valid chart v1",
			content: `apiVersion: v1
name: legacy-chart
version: 0.1.0
description: A legacy chart
`,
			wantErr:     false,
			wantName:    "legacy-chart",
			wantVersion: "0.1.0",
		},
		{
			name: "chart with dependencies",
			content: `apiVersion: v2
name: app-chart
version: 2.0.0
description: Chart with dependencies
dependencies:
  - name: redis
    version: "17.0.0"
    repository: https://charts.bitnami.com/bitnami
  - name: postgresql
    version: "12.0.0"
    repository: https://charts.bitnami.com/bitnami
    condition: postgresql.enabled
`,
			wantErr:     false,
			wantName:    "app-chart",
			wantVersion: "2.0.0",
		},
		{
			name:    "invalid yaml",
			content: `name: [invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			chartFile := filepath.Join(tempDir, "Chart.yaml")

			if err := os.WriteFile(chartFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			chart, err := ParseChart(tempDir)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if chart.Name != tt.wantName {
				t.Errorf("expected name %s, got %s", tt.wantName, chart.Name)
			}
			if chart.Version != tt.wantVersion {
				t.Errorf("expected version %s, got %s", tt.wantVersion, chart.Version)
			}
		})
	}
}

func TestUpdateChartVersion(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		version    string
		appVersion string
		wantErr    bool
		validate   func(t *testing.T, content string)
	}{
		{
			name: "update version only",
			content: `apiVersion: v2
name: my-chart
version: 1.0.0
description: Test chart
`,
			version:    "2.0.0",
			appVersion: "",
			wantErr:    false,
			validate: func(t *testing.T, content string) {
				if !contains(content, "version: 2.0.0") {
					t.Error("version not updated")
				}
			},
		},
		{
			name: "update version and appVersion",
			content: `apiVersion: v2
name: my-chart
version: 1.0.0
appVersion: "1.0.0"
description: Test chart
`,
			version:    "2.0.0",
			appVersion: "2.0.0",
			wantErr:    false,
			validate: func(t *testing.T, content string) {
				if !contains(content, "version: 2.0.0") {
					t.Error("version not updated")
				}
				if !contains(content, `appVersion: "2.0.0"`) {
					t.Error("appVersion not updated")
				}
			},
		},
		{
			name: "add appVersion when missing",
			content: `apiVersion: v2
name: my-chart
version: 1.0.0
description: Test chart
`,
			version:    "2.0.0",
			appVersion: "2.0.0",
			wantErr:    false,
			validate: func(t *testing.T, content string) {
				if !contains(content, "version: 2.0.0") {
					t.Error("version not updated")
				}
				if !contains(content, `appVersion: "2.0.0"`) {
					t.Error("appVersion not added")
				}
			},
		},
		{
			name: "preserve comments",
			content: `apiVersion: v2
name: my-chart
# This is the version
version: 1.0.0
description: Test chart
`,
			version:    "3.0.0",
			appVersion: "",
			wantErr:    false,
			validate: func(t *testing.T, content string) {
				if !contains(content, "version: 3.0.0") {
					t.Error("version not updated")
				}
				if !contains(content, "# This is the version") {
					t.Error("comment not preserved")
				}
			},
		},
		{
			name: "no version field",
			content: `apiVersion: v2
name: my-chart
description: Test chart
`,
			version:    "1.0.0",
			appVersion: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			chartFile := filepath.Join(tempDir, "Chart.yaml")

			if err := os.WriteFile(chartFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}

			err := UpdateChartVersion(tempDir, tt.version, tt.appVersion)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result, err := os.ReadFile(chartFile)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, string(result))
			}
		})
	}
}

func TestValidateChart(t *testing.T) {
	tests := []struct {
		name    string
		chart   *Chart
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid chart",
			chart: &Chart{
				APIVersion: "v2",
				Name:       "my-chart",
				Version:    "1.0.0",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			chart: &Chart{
				APIVersion: "v2",
				Version:    "1.0.0",
			},
			wantErr: true,
			errMsg:  "chart name is required",
		},
		{
			name: "missing version",
			chart: &Chart{
				APIVersion: "v2",
				Name:       "my-chart",
			},
			wantErr: true,
			errMsg:  "chart version is required",
		},
		{
			name: "missing apiVersion",
			chart: &Chart{
				Name:    "my-chart",
				Version: "1.0.0",
			},
			wantErr: true,
			errMsg:  "apiVersion is required",
		},
		{
			name: "invalid apiVersion",
			chart: &Chart{
				APIVersion: "v3",
				Name:       "my-chart",
				Version:    "1.0.0",
			},
			wantErr: true,
			errMsg:  "apiVersion must be v1 or v2, got: v3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChart(tt.chart)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestChartMethods(t *testing.T) {
	chart := &Chart{
		Name:    "test-chart",
		Version: "1.2.3",
		Dependencies: []ChartDependency{
			{Name: "redis", Version: "1.0.0"},
		},
	}

	if chart.GetChartName() != "test-chart" {
		t.Errorf("expected name 'test-chart', got '%s'", chart.GetChartName())
	}

	if chart.GetChartVersion() != "1.2.3" {
		t.Errorf("expected version '1.2.3', got '%s'", chart.GetChartVersion())
	}

	if !chart.HasDependencies() {
		t.Error("expected HasDependencies to be true")
	}

	chartNoDeps := &Chart{Name: "no-deps"}
	if chartNoDeps.HasDependencies() {
		t.Error("expected HasDependencies to be false")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
