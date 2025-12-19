package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewRepository(t *testing.T) {
	config := RepositoryConfig{
		Type:     "oci",
		URL:      "oci://ghcr.io/myorg/charts",
		Username: "user",
		Password: "pass",
	}

	repo := NewRepository(config)

	if repo.config.Type != "oci" {
		t.Errorf("expected type 'oci', got '%s'", repo.config.Type)
	}
	if repo.config.URL != "oci://ghcr.io/myorg/charts" {
		t.Errorf("expected URL 'oci://ghcr.io/myorg/charts', got '%s'", repo.config.URL)
	}
}

func TestRepositorySetContextPath(t *testing.T) {
	repo := NewRepository(RepositoryConfig{})
	repo.SetContextPath("api/v1")

	if repo.contextPath != "api/v1" {
		t.Errorf("expected contextPath 'api/v1', got '%s'", repo.contextPath)
	}
}

func TestRepositoryPushChartMuseum(t *testing.T) {
	// Create a test server
	var receivedAuth string
	var receivedContentType string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/charts" {
			t.Errorf("expected path /api/charts, got %s", r.URL.Path)
		}

		receivedAuth = r.Header.Get("Authorization")
		receivedContentType = r.Header.Get("Content-Type")

		body, _ := io.ReadAll(r.Body)
		receivedBody = body

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"saved": true}`))
	}))
	defer server.Close()

	// Create a test package file
	tempDir := t.TempDir()
	packagePath := filepath.Join(tempDir, "test-chart-1.0.0.tgz")
	testContent := []byte("fake chart content")
	if err := os.WriteFile(packagePath, testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	repo := NewRepository(RepositoryConfig{
		Type:     "chartmuseum",
		URL:      server.URL,
		Username: "testuser",
		Password: "testpass",
	})

	err := repo.Push(context.Background(), packagePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedContentType != "application/gzip" {
		t.Errorf("expected Content-Type 'application/gzip', got '%s'", receivedContentType)
	}

	if receivedAuth == "" {
		t.Error("expected Authorization header to be set")
	}

	if string(receivedBody) != string(testContent) {
		t.Error("body content mismatch")
	}
}

func TestRepositoryPushChartMuseumWithContextPath(t *testing.T) {
	var receivedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	packagePath := filepath.Join(tempDir, "test-chart-1.0.0.tgz")
	if err := os.WriteFile(packagePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	repo := NewRepository(RepositoryConfig{
		Type: "chartmuseum",
		URL:  server.URL,
	})
	repo.SetContextPath("v1")

	err := repo.Push(context.Background(), packagePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedPath != "/v1/api/charts" {
		t.Errorf("expected path '/v1/api/charts', got '%s'", receivedPath)
	}
}

func TestRepositoryPushHTTP(t *testing.T) {
	var receivedMethod string
	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	packagePath := filepath.Join(tempDir, "test-chart-1.0.0.tgz")
	if err := os.WriteFile(packagePath, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	repo := NewRepository(RepositoryConfig{
		Type: "http",
		URL:  server.URL,
	})

	err := repo.Push(context.Background(), packagePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedMethod != "PUT" {
		t.Errorf("expected method PUT, got %s", receivedMethod)
	}

	if receivedContentType != "application/gzip" {
		t.Errorf("expected Content-Type 'application/gzip', got '%s'", receivedContentType)
	}
}

func TestRepositoryPushUnsupportedType(t *testing.T) {
	repo := NewRepository(RepositoryConfig{
		Type: "unknown",
		URL:  "http://example.com",
	})

	err := repo.Push(context.Background(), "/fake/path.tgz")
	if err == nil {
		t.Error("expected error for unsupported type")
	}

	expectedErr := "unsupported repository type: unknown"
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestRepositoryPushChartMuseumError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	packagePath := filepath.Join(tempDir, "test-chart-1.0.0.tgz")
	if err := os.WriteFile(packagePath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	repo := NewRepository(RepositoryConfig{
		Type: "chartmuseum",
		URL:  server.URL,
	})

	err := repo.Push(context.Background(), packagePath)
	if err == nil {
		t.Error("expected error for server error response")
	}
}

func TestRepositoryPushFileNotFound(t *testing.T) {
	repo := NewRepository(RepositoryConfig{
		Type: "chartmuseum",
		URL:  "http://example.com",
	})

	err := repo.Push(context.Background(), "/nonexistent/path.tgz")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
