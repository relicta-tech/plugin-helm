package main

import (
	"testing"

	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

func TestGetInfo(t *testing.T) {
	p := &HelmPlugin{}
	info := p.GetInfo()

	if info.Name != "helm" {
		t.Errorf("expected name 'helm', got '%s'", info.Name)
	}

	if info.Version != Version {
		t.Errorf("expected version '%s', got '%s'", Version, info.Version)
	}

	if len(info.Hooks) != 2 {
		t.Errorf("expected 2 hooks, got %d", len(info.Hooks))
	}

	hasPrePublish := false
	hasPostPublish := false
	for _, hook := range info.Hooks {
		if hook == plugin.HookPrePublish {
			hasPrePublish = true
		}
		if hook == plugin.HookPostPublish {
			hasPostPublish = true
		}
	}

	if !hasPrePublish {
		t.Error("expected PrePublish hook")
	}
	if !hasPostPublish {
		t.Error("expected PostPublish hook")
	}
}

func TestParseConfig(t *testing.T) {
	p := &HelmPlugin{}

	tests := []struct {
		name     string
		raw      map[string]any
		validate func(t *testing.T, cfg *Config)
	}{
		{
			name: "default values",
			raw:  map[string]any{},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.ChartPath != "." {
					t.Errorf("expected default chart_path '.', got '%s'", cfg.ChartPath)
				}
				if !cfg.Lint {
					t.Error("expected lint to be true by default")
				}
				if cfg.LintStrict {
					t.Error("expected lint_strict to be false by default")
				}
				if !cfg.TemplateValidate {
					t.Error("expected template_validate to be true by default")
				}
				if cfg.OutputDir != ".helm-packages" {
					t.Errorf("expected default output_dir '.helm-packages', got '%s'", cfg.OutputDir)
				}
			},
		},
		{
			name: "custom values",
			raw: map[string]any{
				"chart_path":   "./charts/my-app",
				"lint":         false,
				"lint_strict":  true,
				"kube_version": "1.28.0",
				"output_dir":   "./dist",
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.ChartPath != "./charts/my-app" {
					t.Errorf("expected chart_path './charts/my-app', got '%s'", cfg.ChartPath)
				}
				if cfg.Lint {
					t.Error("expected lint to be false")
				}
				if !cfg.LintStrict {
					t.Error("expected lint_strict to be true")
				}
				if cfg.KubeVersion != "1.28.0" {
					t.Errorf("expected kube_version '1.28.0', got '%s'", cfg.KubeVersion)
				}
				if cfg.OutputDir != "./dist" {
					t.Errorf("expected output_dir './dist', got '%s'", cfg.OutputDir)
				}
			},
		},
		{
			name: "repository config",
			raw: map[string]any{
				"repository": map[string]any{
					"type":     "chartmuseum",
					"url":      "https://charts.example.com",
					"name":     "myrepo",
					"username": "user",
					"password": "pass",
				},
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Repository.Type != "chartmuseum" {
					t.Errorf("expected repository type 'chartmuseum', got '%s'", cfg.Repository.Type)
				}
				if cfg.Repository.URL != "https://charts.example.com" {
					t.Errorf("expected repository URL 'https://charts.example.com', got '%s'", cfg.Repository.URL)
				}
				if cfg.Repository.Name != "myrepo" {
					t.Errorf("expected repository name 'myrepo', got '%s'", cfg.Repository.Name)
				}
			},
		},
		{
			name: "version config",
			raw: map[string]any{
				"version": map[string]any{
					"update_chart":       true,
					"update_app_version": true,
					"app_version_format": "v{{.Version}}",
				},
			},
			validate: func(t *testing.T, cfg *Config) {
				if !cfg.Version.UpdateChart {
					t.Error("expected update_chart to be true")
				}
				if !cfg.Version.UpdateAppVersion {
					t.Error("expected update_app_version to be true")
				}
				if cfg.Version.AppVersionFormat != "v{{.Version}}" {
					t.Errorf("expected app_version_format 'v{{.Version}}', got '%s'", cfg.Version.AppVersionFormat)
				}
			},
		},
		{
			name: "dependency config",
			raw: map[string]any{
				"dependencies": map[string]any{
					"update": false,
					"build":  true,
				},
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Dependencies.Update {
					t.Error("expected dependencies.update to be false")
				}
				if !cfg.Dependencies.Build {
					t.Error("expected dependencies.build to be true")
				}
			},
		},
		{
			name: "signing config",
			raw: map[string]any{
				"sign":            true,
				"sign_key":        "mykey",
				"keyring":         "/path/to/keyring",
				"passphrase_file": "/path/to/passphrase",
			},
			validate: func(t *testing.T, cfg *Config) {
				if !cfg.Sign {
					t.Error("expected sign to be true")
				}
				if cfg.SignKey != "mykey" {
					t.Errorf("expected sign_key 'mykey', got '%s'", cfg.SignKey)
				}
				if cfg.Keyring != "/path/to/keyring" {
					t.Errorf("expected keyring '/path/to/keyring', got '%s'", cfg.Keyring)
				}
				if cfg.PassphraseFile != "/path/to/passphrase" {
					t.Errorf("expected passphrase_file '/path/to/passphrase', got '%s'", cfg.PassphraseFile)
				}
			},
		},
		{
			name: "api versions",
			raw: map[string]any{
				"api_versions": []any{
					"apps/v1",
					"batch/v1",
					"networking.k8s.io/v1",
				},
			},
			validate: func(t *testing.T, cfg *Config) {
				if len(cfg.APIVersions) != 3 {
					t.Errorf("expected 3 api_versions, got %d", len(cfg.APIVersions))
				}
				expected := []string{"apps/v1", "batch/v1", "networking.k8s.io/v1"}
				for i, v := range expected {
					if cfg.APIVersions[i] != v {
						t.Errorf("expected api_version[%d] '%s', got '%s'", i, v, cfg.APIVersions[i])
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := p.parseConfig(tt.raw)
			tt.validate(t, cfg)
		})
	}
}

func TestVersionConfigDefaults(t *testing.T) {
	p := &HelmPlugin{}
	cfg := p.parseConfig(map[string]any{})

	// Version config defaults
	if !cfg.Version.UpdateChart {
		t.Error("expected version.update_chart to be true by default")
	}
	if !cfg.Version.UpdateAppVersion {
		t.Error("expected version.update_app_version to be true by default")
	}
}

func TestDependencyConfigDefaults(t *testing.T) {
	p := &HelmPlugin{}
	cfg := p.parseConfig(map[string]any{})

	// Dependency config defaults
	if !cfg.Dependencies.Update {
		t.Error("expected dependencies.update to be true by default")
	}
	if !cfg.Dependencies.Build {
		t.Error("expected dependencies.build to be true by default")
	}
}

func TestRepositoryConfigDefaults(t *testing.T) {
	p := &HelmPlugin{}
	cfg := p.parseConfig(map[string]any{})

	// Repository config defaults
	if cfg.Repository.Type != "oci" {
		t.Errorf("expected repository.type to be 'oci' by default, got '%s'", cfg.Repository.Type)
	}
}
