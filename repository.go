package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Repository handles chart repository operations.
type Repository struct {
	config      RepositoryConfig
	contextPath string
}

// NewRepository creates a new repository handler.
func NewRepository(config RepositoryConfig) *Repository {
	return &Repository{
		config: config,
	}
}

// SetContextPath sets the context path for ChartMuseum.
func (r *Repository) SetContextPath(path string) {
	r.contextPath = path
}

// Push pushes a chart to the repository.
func (r *Repository) Push(ctx context.Context, packagePath string) error {
	switch r.config.Type {
	case "oci":
		return r.pushOCI(ctx, packagePath)
	case "chartmuseum":
		return r.pushChartMuseum(ctx, packagePath)
	case "http":
		return r.pushHTTP(ctx, packagePath)
	default:
		return fmt.Errorf("unsupported repository type: %s", r.config.Type)
	}
}

// pushOCI pushes to an OCI registry.
func (r *Repository) pushOCI(ctx context.Context, packagePath string) error {
	// Login to registry if credentials provided
	if r.config.Username != "" && r.config.Password != "" {
		registry := strings.TrimPrefix(r.config.URL, "oci://")
		// Extract just the host part
		parts := strings.SplitN(registry, "/", 2)
		registryHost := parts[0]

		if err := r.registryLogin(ctx, registryHost); err != nil {
			return fmt.Errorf("registry login failed: %w", err)
		}
	}

	// Push chart
	cmd := exec.CommandContext(ctx, "helm", "push", packagePath, r.config.URL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm push failed: %w", err)
	}

	return nil
}

// pushChartMuseum pushes to ChartMuseum.
func (r *Repository) pushChartMuseum(ctx context.Context, packagePath string) error {
	file, err := os.Open(packagePath)
	if err != nil {
		return fmt.Errorf("failed to open package: %w", err)
	}
	defer func() { _ = file.Close() }()

	endpoint := r.config.URL + "/api/charts"
	if r.contextPath != "" {
		endpoint = r.config.URL + "/" + r.contextPath + "/api/charts"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/gzip")

	if r.config.Username != "" && r.config.Password != "" {
		req.SetBasicAuth(r.config.Username, r.config.Password)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to push chart: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// pushHTTP pushes to an HTTP repository (generic upload).
func (r *Repository) pushHTTP(ctx context.Context, packagePath string) error {
	file, err := os.Open(packagePath)
	if err != nil {
		return fmt.Errorf("failed to open package: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Get file info for Content-Length
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat package: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", r.config.URL, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/gzip")
	req.ContentLength = stat.Size()

	if r.config.Username != "" && r.config.Password != "" {
		req.SetBasicAuth(r.config.Username, r.config.Password)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to push chart: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// registryLogin performs registry login for OCI.
func (r *Repository) registryLogin(ctx context.Context, registry string) error {
	cmd := exec.CommandContext(ctx, "helm", "registry", "login", registry,
		"--username", r.config.Username,
		"--password-stdin")
	cmd.Stdin = strings.NewReader(r.config.Password)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Logout performs registry logout for OCI.
func (r *Repository) Logout(ctx context.Context) error {
	if r.config.Type != "oci" {
		return nil
	}

	registry := strings.TrimPrefix(r.config.URL, "oci://")
	parts := strings.SplitN(registry, "/", 2)
	registryHost := parts[0]

	cmd := exec.CommandContext(ctx, "helm", "registry", "logout", registryHost)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
