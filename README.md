# plugin-helm

Relicta plugin for publishing Helm charts to OCI registries and chart repositories.

## Features

- Publish charts to OCI registries (GHCR, ECR, ACR, Docker Hub)
- Push to ChartMuseum instances
- Push to HTTP repositories (Nexus, Artifactory)
- Automatic version updates in Chart.yaml
- Chart linting and template validation
- Dependency management
- Optional chart signing

## Installation

```bash
relicta plugin install helm
```

## Configuration

```yaml
plugins:
  - name: helm
    enabled: true
    hooks:
      - PrePublish
      - PostPublish
    config:
      # Chart directory
      chart_path: "./charts/my-app"

      # Repository configuration
      repository:
        type: "oci"  # oci, http, chartmuseum
        url: "oci://ghcr.io/myorg/charts"
        username: ${HELM_REPO_USERNAME}
        password: ${HELM_REPO_PASSWORD}

      # Version management
      version:
        update_chart: true
        update_app_version: true
        app_version_format: "{{.Version}}"

      # Validation
      lint: true
      lint_strict: false
      template_validate: true
      kube_version: "1.28.0"

      # Dependencies
      dependencies:
        update: true
        build: true

      # Signing (optional)
      sign: false
      sign_key: ""
      keyring: ""

      # Output
      output_dir: ".helm-packages"
```

## Repository Types

### OCI Registry

For OCI-compliant registries like GitHub Container Registry, AWS ECR, or Azure ACR:

```yaml
repository:
  type: "oci"
  url: "oci://ghcr.io/myorg/charts"
  username: ${GITHUB_ACTOR}
  password: ${GITHUB_TOKEN}
```

### ChartMuseum

For ChartMuseum instances:

```yaml
repository:
  type: "chartmuseum"
  url: "https://chartmuseum.example.com"
  username: ${CHARTMUSEUM_USER}
  password: ${CHARTMUSEUM_PASSWORD}
```

### HTTP Repository

For generic HTTP repositories (Nexus, Artifactory):

```yaml
repository:
  type: "http"
  url: "https://nexus.example.com/repository/helm-releases/my-chart-1.0.0.tgz"
  username: ${NEXUS_USER}
  password: ${NEXUS_PASSWORD}
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `HELM_REPO_USERNAME` | Repository username |
| `HELM_REPO_PASSWORD` | Repository password or token |
| `HELM_REGISTRY_CONFIG` | Path to Docker config for OCI auth |

## Hooks

### PrePublish

Runs before the release is published:
- Updates version in Chart.yaml
- Updates chart dependencies
- Lints the chart
- Validates templates

### PostPublish

Runs after the release is published:
- Packages the chart
- Signs the package (if enabled)
- Pushes to the repository

## Chart Signing

To sign charts with GPG:

```yaml
config:
  sign: true
  sign_key: "your-gpg-key-id"
  keyring: "~/.gnupg/pubring.gpg"
  passphrase_file: "/path/to/passphrase"
```

## Dry Run

Test the plugin without making changes:

```bash
relicta publish --dry-run
```

## Requirements

- Helm 3.x (required for OCI support)
- For OCI: Docker credentials configured
- For signing: GPG key available

## Development

```bash
# Build
go build -o plugin-helm

# Test
go test -v ./...

# Lint
golangci-lint run
```

## License

MIT License - see [LICENSE](LICENSE) for details.
