# prometheus-opentelemetry-collector

An OpenTelemetry Collector distro, by Prometheus.

This repository contains a custom distribution of the OpenTelemetry Collector, built using the [OpenTelemetry Collector Builder (OCB)](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder). It includes Prometheus-specific receivers and exporters.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Makefile Targets](#makefile-targets)
- [Building the Collector](#building-the-collector)
- [Configuration](#configuration)
- [Customizing Components](#customizing-components)
- [Development Workflow](#development-workflow)

## Prerequisites

Before building the custom collector, ensure you have the following installed:

1. **Go** (version 1.21 or later)
   ```bash
   go version
   ```

2. **OpenTelemetry Collector Builder (OCB)**
   ```bash
   go install go.opentelemetry.io/collector/cmd/builder@latest
   ```

   The builder will be installed in `~/go/bin/builder`. Make sure `~/go/bin` is in your `PATH`:
   ```bash
   export PATH=$PATH:~/go/bin
   ```

You can verify OCB is installed correctly:
```bash
make check-ocb
```

## Quick Start

1. **Build the collector**:
   ```bash
   make build
   ```
   
   This will:
   - Verify OCB is installed
   - Generate the collector based on `builder-config.yaml`
   - Compile the binary to `./dist/prometheus-opentelemetry-collector`

2. **Run the collector**:
   ```bash
   make run
   ```
   
   This starts the collector using the configuration in `otel-config.yaml`.

3. **Validate your configuration**:
   ```bash
   make validate
   ```

### `make validate`
Validate the collector configuration file (`otel-config.yaml`) without starting the collector.

```bash
make validate
```

This is useful for checking configuration syntax and component compatibility before running.

### `make components`
List all components (receivers, processors, exporters, extensions) included in the built collector.

```bash
make components
```

This helps verify which components are available in your custom build.

## Building the Collector

The build process uses the `builder-config.yaml` file, which defines:

- **Distribution name and version**
- **OpenTelemetry Collector core version**
- **Components to include** (receivers, processors, exporters, extensions)
- **Module replacements** for local development

## Customizing Components

To add or remove components from your custom collector:

1. **Edit `builder-config.yaml`** to add/remove components:

   ```yaml
   exporters:
     - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.138.0
     - gomod: go.opentelemetry.io/collector/exporter/otlpexporter v0.138.0
   
   receivers:
     - gomod: github.com/prometheus/node_exporter/otelcollector v0.1.0
     - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.138.0
   
   processors:
     - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.138.0
   ```

2. **Rebuild the collector**:
   ```bash
   make rebuild
   ```

3. **Verify components are available**:
   ```bash
   make components
   ```

Find available components in the [OpenTelemetry Collector Contrib repository](https://github.com/open-telemetry/opentelemetry-collector-contrib).

## Development Workflow

### Typical workflow:

1. **Make changes** to `builder-config.yaml` or update dependencies
2. **Rebuild** the collector:
   ```bash
   make rebuild
   ```
3. **Validate** your runtime configuration:
   ```bash
   make validate
   ```
4. **Run** the collector:
   ```bash
   make run
   ```
5. **Inspect** included components:
   ```bash
   make components
   ```

### When using local module replacements:

The `replaces` section in `builder-config.yaml` allows you to use local versions of modules for development:

```yaml
replaces:
  - github.com/prometheus/node_exporter => ../../node_exporter
```

This is useful when:
- Developing new components locally
- Testing unreleased changes
- Debugging issues in dependencies

Make sure the local paths exist relative to the build directory.

## License

See [LICENSE](LICENSE) file for details.
