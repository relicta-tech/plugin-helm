# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2024-12-19

### Added

- Initial release
- Support for OCI registries (GHCR, ECR, ACR, Docker Hub)
- Support for ChartMuseum repositories
- Support for HTTP repositories (Nexus, Artifactory)
- Chart.yaml version and appVersion management
- Chart linting with configurable strictness
- Template validation with Kubernetes version targeting
- Chart dependency update and build
- Optional GPG chart signing
- Dry-run mode for testing
- PrePublish hook for validation and version updates
- PostPublish hook for packaging and pushing
