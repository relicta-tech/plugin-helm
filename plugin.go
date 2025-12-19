package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/relicta-tech/relicta-plugin-sdk/helpers"
	"github.com/relicta-tech/relicta-plugin-sdk/plugin"
)

// Version is set at build time.
var Version = "0.1.0"

// Config represents Helm plugin configuration.
type Config struct {
	ChartPath        string           `json:"chart_path"`
	Repository       RepositoryConfig `json:"repository"`
	Version          VersionConfig    `json:"version"`
	Lint             bool             `json:"lint"`
	LintStrict       bool             `json:"lint_strict"`
	TemplateValidate bool             `json:"template_validate"`
	Test             bool             `json:"test"`
	KubeVersion      string           `json:"kube_version"`
	APIVersions      []string         `json:"api_versions"`
	Dependencies     DependencyConfig `json:"dependencies"`
	Sign             bool             `json:"sign"`
	SignKey          string           `json:"sign_key"`
	Keyring          string           `json:"keyring"`
	PassphraseFile   string           `json:"passphrase_file"`
	OutputDir        string           `json:"output_dir"`
	ContextPath      string           `json:"context_path"`
	DryRun           bool             `json:"dry_run"`
}

// RepositoryConfig defines repository settings.
type RepositoryConfig struct {
	Type           string `json:"type"` // oci, http, chartmuseum
	URL            string `json:"url"`
	Name           string `json:"name"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	RegistryConfig string `json:"registry_config"`
}

// VersionConfig defines version update settings.
type VersionConfig struct {
	UpdateChart      bool   `json:"update_chart"`
	UpdateAppVersion bool   `json:"update_app_version"`
	AppVersionFormat string `json:"app_version_format"`
}

// DependencyConfig defines dependency management settings.
type DependencyConfig struct {
	Update bool `json:"update"`
	Build  bool `json:"build"`
}

// HelmPlugin implements the Helm chart plugin.
type HelmPlugin struct{}

// GetInfo returns plugin metadata.
func (p *HelmPlugin) GetInfo() plugin.Info {
	return plugin.Info{
		Name:        "helm",
		Version:     Version,
		Description: "Helm chart publishing to OCI registries and chart repositories",
		Hooks: []plugin.Hook{
			plugin.HookPrePublish,
			plugin.HookPostPublish,
		},
	}
}

// Validate validates plugin configuration.
func (p *HelmPlugin) Validate(ctx context.Context, config map[string]any) (*plugin.ValidateResponse, error) {
	cfg := p.parseConfig(config)
	vb := helpers.NewValidationBuilder()

	// Check Helm installation
	helmVersion, err := getHelmVersion()
	if err != nil {
		vb.AddError("helm", "Helm CLI not found in PATH")
	} else if !strings.HasPrefix(helmVersion, "v3") {
		vb.AddError("helm", "Helm 3.x required for OCI support")
	}

	// Check chart exists
	chartPath := cfg.ChartPath
	if chartPath == "" {
		chartPath = "."
	}

	chartFile := filepath.Join(chartPath, "Chart.yaml")
	if _, err := os.Stat(chartFile); os.IsNotExist(err) {
		vb.AddError("chart_path", "Chart.yaml not found")
	} else {
		// Validate chart
		chart, err := ParseChart(chartPath)
		if err != nil {
			vb.AddError("chart_path", fmt.Sprintf("Invalid Chart.yaml: %v", err))
		} else if chart.Name == "" {
			vb.AddError("chart_path", "Chart name is required")
		}
	}

	// Check repository configuration
	if cfg.Repository.URL == "" {
		vb.AddError("repository.url", "Repository URL is required")
	}

	// For OCI, verify Helm version supports it
	if cfg.Repository.Type == "oci" && err == nil && !strings.HasPrefix(helmVersion, "v3") {
		vb.AddError("repository.type", "OCI requires Helm 3.x")
	}

	return vb.Build(), nil
}

// Execute runs the plugin for a given hook.
func (p *HelmPlugin) Execute(ctx context.Context, req plugin.ExecuteRequest) (*plugin.ExecuteResponse, error) {
	cfg := p.parseConfig(req.Config)
	cfg.DryRun = cfg.DryRun || req.DryRun
	logger := slog.Default().With("plugin", "helm", "hook", req.Hook)

	switch req.Hook {
	case plugin.HookPrePublish:
		return p.executePrePublish(ctx, &req.Context, cfg, logger)
	case plugin.HookPostPublish:
		return p.executePostPublish(ctx, &req.Context, cfg, logger)
	default:
		return &plugin.ExecuteResponse{
			Success: true,
			Message: fmt.Sprintf("Hook %s not handled by helm plugin", req.Hook),
		}, nil
	}
}

func (p *HelmPlugin) executePrePublish(ctx context.Context, releaseCtx *plugin.ReleaseContext, cfg *Config, logger *slog.Logger) (*plugin.ExecuteResponse, error) {
	version := releaseCtx.Version
	logger = logger.With("version", version)

	chartPath := cfg.ChartPath
	if chartPath == "" {
		chartPath = "."
	}

	// Parse chart to get name
	chart, err := ParseChart(chartPath)
	if err != nil {
		return &plugin.ExecuteResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to parse Chart.yaml: %v", err),
		}, nil
	}

	logger = logger.With("chart", chart.Name)

	helm := NewHelmCLI(chartPath)

	// Update version in Chart.yaml
	if cfg.Version.UpdateChart {
		logger.Info("Updating version in Chart.yaml")
		appVersion := ""
		if cfg.Version.UpdateAppVersion {
			appVersion = version
			if cfg.Version.AppVersionFormat != "" {
				appVersion = strings.ReplaceAll(cfg.Version.AppVersionFormat, "{{.Version}}", version)
			}
		}

		if cfg.DryRun {
			logger.Info("[DRY-RUN] Would update Chart.yaml", "from", chart.Version, "to", version, "appVersion", appVersion)
		} else {
			if err := UpdateChartVersion(chartPath, version, appVersion); err != nil {
				return &plugin.ExecuteResponse{
					Success: false,
					Message: fmt.Sprintf("Failed to update Chart.yaml version: %v", err),
				}, nil
			}
		}
	}

	// Update dependencies
	if cfg.Dependencies.Update {
		logger.Info("Updating chart dependencies")
		if cfg.DryRun {
			logger.Info("[DRY-RUN] Would run helm dependency update")
		} else {
			if err := helm.DependencyUpdate(ctx); err != nil {
				return &plugin.ExecuteResponse{
					Success: false,
					Message: fmt.Sprintf("Failed to update dependencies: %v", err),
				}, nil
			}
		}
	}

	// Build dependencies
	if cfg.Dependencies.Build {
		logger.Info("Building chart dependencies")
		if cfg.DryRun {
			logger.Info("[DRY-RUN] Would run helm dependency build")
		} else {
			if err := helm.DependencyBuild(ctx); err != nil {
				return &plugin.ExecuteResponse{
					Success: false,
					Message: fmt.Sprintf("Failed to build dependencies: %v", err),
				}, nil
			}
		}
	}

	// Lint chart
	if cfg.Lint {
		logger.Info("Linting chart", "strict", cfg.LintStrict)
		if cfg.DryRun {
			logger.Info("[DRY-RUN] Would run helm lint")
		} else {
			if err := helm.Lint(ctx, cfg.LintStrict); err != nil {
				return &plugin.ExecuteResponse{
					Success: false,
					Message: fmt.Sprintf("Chart linting failed: %v", err),
				}, nil
			}
		}
	}

	// Template validation
	if cfg.TemplateValidate {
		logger.Info("Validating chart templates", "kubeVersion", cfg.KubeVersion)
		if cfg.DryRun {
			logger.Info("[DRY-RUN] Would run helm template validation")
		} else {
			if err := helm.Template(ctx, cfg.KubeVersion, cfg.APIVersions); err != nil {
				return &plugin.ExecuteResponse{
					Success: false,
					Message: fmt.Sprintf("Template validation failed: %v", err),
				}, nil
			}
		}
	}

	logger.Info("PrePublish completed successfully")
	return &plugin.ExecuteResponse{
		Success: true,
		Message: fmt.Sprintf("Chart %s validated successfully", chart.Name),
	}, nil
}

func (p *HelmPlugin) executePostPublish(ctx context.Context, releaseCtx *plugin.ReleaseContext, cfg *Config, logger *slog.Logger) (*plugin.ExecuteResponse, error) {
	version := releaseCtx.Version
	logger = logger.With("version", version)

	chartPath := cfg.ChartPath
	if chartPath == "" {
		chartPath = "."
	}

	// Parse chart to get name
	chart, err := ParseChart(chartPath)
	if err != nil {
		return &plugin.ExecuteResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to parse Chart.yaml: %v", err),
		}, nil
	}

	logger = logger.With("chart", chart.Name)

	helm := NewHelmCLI(chartPath)

	// Ensure output directory exists
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = ".helm-packages"
	}

	if !cfg.DryRun {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return &plugin.ExecuteResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to create output directory: %v", err),
			}, nil
		}
	}

	// Package chart
	logger.Info("Packaging chart", "outputDir", outputDir)
	var packagePath string
	if cfg.DryRun {
		logger.Info("[DRY-RUN] Would package chart",
			"sign", cfg.Sign,
			"outputDir", outputDir)
		packagePath = filepath.Join(outputDir, fmt.Sprintf("%s-%s.tgz", chart.Name, version))
	} else {
		var signOpts *SignOptions
		if cfg.Sign {
			signOpts = &SignOptions{
				Keyring:        cfg.Keyring,
				Key:            cfg.SignKey,
				PassphraseFile: cfg.PassphraseFile,
			}
		}

		packagePath, err = helm.Package(ctx, outputDir, signOpts)
		if err != nil {
			return &plugin.ExecuteResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to package chart: %v", err),
			}, nil
		}
	}

	// Push to repository
	logger.Info("Pushing chart to repository",
		"type", cfg.Repository.Type,
		"url", cfg.Repository.URL)

	repo := NewRepository(cfg.Repository)
	repo.SetContextPath(cfg.ContextPath)

	if cfg.DryRun {
		logger.Info("[DRY-RUN] Would push chart",
			"package", packagePath,
			"repository", cfg.Repository.URL)
	} else {
		if err := repo.Push(ctx, packagePath); err != nil {
			return &plugin.ExecuteResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to push chart: %v", err),
			}, nil
		}
	}

	var msg string
	if cfg.DryRun {
		msg = fmt.Sprintf("[DRY-RUN] Would publish %s@%s to %s", chart.Name, version, cfg.Repository.URL)
	} else {
		msg = fmt.Sprintf("Published %s@%s to %s", chart.Name, version, cfg.Repository.URL)
	}

	logger.Info("PostPublish completed successfully")
	return &plugin.ExecuteResponse{
		Success: true,
		Message: msg,
	}, nil
}

func (p *HelmPlugin) parseConfig(raw map[string]any) *Config {
	parser := helpers.NewConfigParser(raw)

	// Parse repository config
	repoConfig := RepositoryConfig{
		Type: "oci",
	}
	if repoRaw, ok := raw["repository"].(map[string]any); ok {
		if t, ok := repoRaw["type"].(string); ok {
			repoConfig.Type = t
		}
		if url, ok := repoRaw["url"].(string); ok {
			repoConfig.URL = url
		}
		if name, ok := repoRaw["name"].(string); ok {
			repoConfig.Name = name
		}
		if username, ok := repoRaw["username"].(string); ok {
			repoConfig.Username = username
		}
		if password, ok := repoRaw["password"].(string); ok {
			repoConfig.Password = password
		}
		if regConfig, ok := repoRaw["registry_config"].(string); ok {
			repoConfig.RegistryConfig = regConfig
		}
	}

	// Parse version config
	versionConfig := VersionConfig{
		UpdateChart:      true,
		UpdateAppVersion: true,
	}
	if versionRaw, ok := raw["version"].(map[string]any); ok {
		if update, ok := versionRaw["update_chart"].(bool); ok {
			versionConfig.UpdateChart = update
		}
		if updateApp, ok := versionRaw["update_app_version"].(bool); ok {
			versionConfig.UpdateAppVersion = updateApp
		}
		if format, ok := versionRaw["app_version_format"].(string); ok {
			versionConfig.AppVersionFormat = format
		}
	}

	// Parse dependency config
	depConfig := DependencyConfig{
		Update: true,
		Build:  true,
	}
	if depRaw, ok := raw["dependencies"].(map[string]any); ok {
		if update, ok := depRaw["update"].(bool); ok {
			depConfig.Update = update
		}
		if build, ok := depRaw["build"].(bool); ok {
			depConfig.Build = build
		}
	}

	// Parse API versions
	var apiVersions []string
	if apiRaw, ok := raw["api_versions"].([]any); ok {
		for _, v := range apiRaw {
			if s, ok := v.(string); ok {
				apiVersions = append(apiVersions, s)
			}
		}
	}

	return &Config{
		ChartPath:        parser.GetString("chart_path", "", "."),
		Repository:       repoConfig,
		Version:          versionConfig,
		Lint:             parser.GetBool("lint", true),
		LintStrict:       parser.GetBool("lint_strict", false),
		TemplateValidate: parser.GetBool("template_validate", true),
		Test:             parser.GetBool("test", false),
		KubeVersion:      parser.GetString("kube_version", "", ""),
		APIVersions:      apiVersions,
		Dependencies:     depConfig,
		Sign:             parser.GetBool("sign", false),
		SignKey:          parser.GetString("sign_key", "", ""),
		Keyring:          parser.GetString("keyring", "", ""),
		PassphraseFile:   parser.GetString("passphrase_file", "", ""),
		OutputDir:        parser.GetString("output_dir", "", ".helm-packages"),
		ContextPath:      parser.GetString("context_path", "", ""),
		DryRun:           parser.GetBool("dry_run", false),
	}
}

func getHelmVersion() (string, error) {
	cmd := exec.Command("helm", "version", "--short")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
