package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// HelmCLI wraps Helm command-line operations.
type HelmCLI struct {
	chartPath string
}

// SignOptions contains chart signing options.
type SignOptions struct {
	Keyring        string
	Key            string
	PassphraseFile string
}

// NewHelmCLI creates a new Helm CLI wrapper.
func NewHelmCLI(chartPath string) *HelmCLI {
	return &HelmCLI{
		chartPath: chartPath,
	}
}

// Lint lints the chart.
func (h *HelmCLI) Lint(ctx context.Context, strict bool) error {
	args := []string{"lint", h.chartPath}
	if strict {
		args = append(args, "--strict")
	}
	return h.run(ctx, args...)
}

// Template validates templates by rendering them.
func (h *HelmCLI) Template(ctx context.Context, kubeVersion string, apiVersions []string) error {
	args := []string{"template", "release-name", h.chartPath}
	if kubeVersion != "" {
		args = append(args, "--kube-version", kubeVersion)
	}
	for _, api := range apiVersions {
		args = append(args, "--api-versions", api)
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = io.Discard // We just want to validate, not see output
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// DependencyUpdate updates chart dependencies.
func (h *HelmCLI) DependencyUpdate(ctx context.Context) error {
	return h.run(ctx, "dependency", "update", h.chartPath)
}

// DependencyBuild builds chart dependencies.
func (h *HelmCLI) DependencyBuild(ctx context.Context) error {
	return h.run(ctx, "dependency", "build", h.chartPath)
}

// Package packages the chart.
func (h *HelmCLI) Package(ctx context.Context, outputDir string, signOpts *SignOptions) (string, error) {
	args := []string{"package", h.chartPath, "-d", outputDir}

	if signOpts != nil {
		args = append(args, "--sign")
		if signOpts.Keyring != "" {
			args = append(args, "--keyring", signOpts.Keyring)
		}
		if signOpts.Key != "" {
			args = append(args, "--key", signOpts.Key)
		}
		if signOpts.PassphraseFile != "" {
			args = append(args, "--passphrase-file", signOpts.PassphraseFile)
		}
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("helm package failed: %w\n%s", err, string(output))
	}

	// Extract package path from output
	// Output: "Successfully packaged chart and saved it to: /path/to/chart-1.0.0.tgz"
	return extractPackagePath(string(output))
}

// Test runs helm test on the chart.
func (h *HelmCLI) Test(ctx context.Context, releaseName string) error {
	return h.run(ctx, "test", releaseName)
}

func (h *HelmCLI) run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// extractPackagePath extracts the package path from helm package output.
func extractPackagePath(output string) (string, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "saved it to:") {
			// Handle the case where path might contain colons (Windows paths)
			idx := strings.Index(line, "saved it to:")
			if idx != -1 {
				path := strings.TrimSpace(line[idx+len("saved it to:"):])
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("could not determine package path from output: %s", output)
}
