package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Chart represents Chart.yaml contents.
type Chart struct {
	APIVersion   string            `yaml:"apiVersion"`
	Name         string            `yaml:"name"`
	Version      string            `yaml:"version"`
	AppVersion   string            `yaml:"appVersion,omitempty"`
	Description  string            `yaml:"description"`
	Type         string            `yaml:"type,omitempty"`
	Keywords     []string          `yaml:"keywords,omitempty"`
	Home         string            `yaml:"home,omitempty"`
	Sources      []string          `yaml:"sources,omitempty"`
	Dependencies []ChartDependency `yaml:"dependencies,omitempty"`
	Maintainers  []Maintainer      `yaml:"maintainers,omitempty"`
	Icon         string            `yaml:"icon,omitempty"`
	Deprecated   bool              `yaml:"deprecated,omitempty"`
	KubeVersion  string            `yaml:"kubeVersion,omitempty"`
}

// ChartDependency represents a chart dependency.
type ChartDependency struct {
	Name       string   `yaml:"name"`
	Version    string   `yaml:"version"`
	Repository string   `yaml:"repository"`
	Condition  string   `yaml:"condition,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
	Alias      string   `yaml:"alias,omitempty"`
}

// Maintainer represents a chart maintainer.
type Maintainer struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email,omitempty"`
	URL   string `yaml:"url,omitempty"`
}

// ParseChart parses a Chart.yaml file.
func ParseChart(chartPath string) (*Chart, error) {
	chartFile := filepath.Join(chartPath, "Chart.yaml")
	data, err := os.ReadFile(chartFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read Chart.yaml: %w", err)
	}

	var chart Chart
	if err := yaml.Unmarshal(data, &chart); err != nil {
		return nil, fmt.Errorf("failed to parse Chart.yaml: %w", err)
	}

	return &chart, nil
}

// UpdateChartVersion updates the version in Chart.yaml.
func UpdateChartVersion(chartPath, version, appVersion string) error {
	chartFile := filepath.Join(chartPath, "Chart.yaml")
	data, err := os.ReadFile(chartFile)
	if err != nil {
		return fmt.Errorf("failed to read Chart.yaml: %w", err)
	}

	// Update version using regex to preserve formatting
	versionPattern := regexp.MustCompile(`(?m)^version:\s*.+$`)
	if !versionPattern.Match(data) {
		return fmt.Errorf("version field not found in Chart.yaml")
	}
	data = versionPattern.ReplaceAll(data, []byte(fmt.Sprintf("version: %s", version)))

	// Update appVersion if provided
	if appVersion != "" {
		appVersionPattern := regexp.MustCompile(`(?m)^appVersion:\s*.+$`)
		if appVersionPattern.Match(data) {
			// Update existing appVersion
			data = appVersionPattern.ReplaceAll(data, []byte(fmt.Sprintf("appVersion: %q", appVersion)))
		} else {
			// Add appVersion after version
			versionLine := regexp.MustCompile(`(?m)^version:\s*.+$`)
			data = versionLine.ReplaceAll(data,
				[]byte(fmt.Sprintf("version: %s\nappVersion: %q", version, appVersion)))
		}
	}

	if err := os.WriteFile(chartFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write Chart.yaml: %w", err)
	}

	return nil
}

// ValidateChart validates Chart.yaml contents.
func ValidateChart(chart *Chart) error {
	if chart.Name == "" {
		return fmt.Errorf("chart name is required")
	}

	if chart.Version == "" {
		return fmt.Errorf("chart version is required")
	}

	// Validate API version
	if chart.APIVersion == "" {
		return fmt.Errorf("apiVersion is required")
	}

	if chart.APIVersion != "v2" && chart.APIVersion != "v1" {
		return fmt.Errorf("apiVersion must be v1 or v2, got: %s", chart.APIVersion)
	}

	return nil
}

// GetChartName returns the chart name.
func (c *Chart) GetChartName() string {
	return c.Name
}

// GetChartVersion returns the chart version.
func (c *Chart) GetChartVersion() string {
	return c.Version
}

// HasDependencies returns true if the chart has dependencies.
func (c *Chart) HasDependencies() bool {
	return len(c.Dependencies) > 0
}
