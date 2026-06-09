// Copyright 2020 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stackdriver

import (
	"context"
	"reflect"
	"testing"

	"github.com/prometheus-community/stackdriver_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestNewFactory(t *testing.T) {
	t.Parallel()

	factory := NewFactory()
	if factory == nil {
		t.Fatal("NewFactory() returned nil")
	}
}

func createMetrics(t *testing.T, exporterConfig map[string]interface{}) error {
	t.Helper()

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*prombridge.ReceiverConfig)
	cfg.ExporterConfig = exporterConfig

	_, err := factory.CreateMetrics(
		context.Background(),
		receivertest.NewNopSettings(receiverType),
		cfg,
		new(consumertest.MetricsSink),
	)
	return err
}

// TestFactory_DefaultConfigDerivesExporterDefaults confirms the bridge derives
// the rendered defaults straight from config.Config: the full map must equal the
// upstream defaults. Breaks if a default value or its rendering changes; field
// add/remove/rename is also covered by TestConfigStructShape.
func TestFactory_DefaultConfigDerivesExporterDefaults(t *testing.T) {
	t.Parallel()

	cfg := NewFactory().CreateDefaultConfig().(*prombridge.ReceiverConfig)

	want := map[string]interface{}{
		"project_ids":                  []string(nil),
		"projects_filter":              "",
		"universe_domain":              config.DefaultUniverseDomain,
		"max_retries":                  config.DefaultMaxRetries,
		"http_timeout":                 config.DefaultHTTPTimeout.String(),
		"max_backoff":                  config.DefaultMaxBackoff.String(),
		"backoff_jitter":               config.DefaultBackoffJitter.String(),
		"retry_statuses":               config.DefaultRetryStatuses,
		"metrics_prefixes":             []string(nil),
		"metrics_interval":             config.DefaultMetricsInterval.String(),
		"metrics_offset":               config.DefaultMetricsOffset.String(),
		"metrics_ingest_delay":         config.DefaultMetricsIngest,
		"fill_missing_labels":          config.DefaultFillMissing,
		"drop_delegated_projects":      config.DefaultDropDelegated,
		"filters":                      []string(nil),
		"aggregate_deltas":             config.DefaultAggregateDeltas,
		"aggregate_deltas_ttl":         config.DefaultDeltasTTL.String(),
		"descriptor_cache_ttl":         config.DefaultDescriptorTTL.String(),
		"descriptor_cache_only_google": config.DefaultDescriptorGoogleOnly,
	}

	if !reflect.DeepEqual(cfg.ExporterConfig, want) {
		t.Fatalf("derived exporter defaults mismatch.\n got: %#v\nwant: %#v", cfg.ExporterConfig, want)
	}
}

func TestFactory_CreateMetrics(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"project_ids":      []string{"my-project"},
		"metrics_prefixes": []string{"compute.googleapis.com/instance"},
	})
	if err != nil {
		t.Fatalf("CreateMetrics() error = %v", err)
	}
}

// TestFactory_CreateMetrics_DecodesUntaggedConfig exercises the bridge wiring:
// the snake_case keys map onto config.Config's fields and the duration strings
// decode through the hook NewFactory installs.
func TestFactory_CreateMetrics_DecodesUntaggedConfig(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"project_ids":                  []string{"my-project"},
		"metrics_prefixes":             []string{"compute.googleapis.com/instance"},
		"http_timeout":                 "30s",
		"metrics_interval":             "10m",
		"aggregate_deltas":             true,
		"aggregate_deltas_ttl":         "1h",
		"descriptor_cache_only_google": false,
	})
	if err != nil {
		t.Fatalf("CreateMetrics() error = %v", err)
	}
}

func TestFactory_CreateMetrics_InvalidDuration(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"metrics_prefixes": []string{"compute.googleapis.com/instance"},
		"http_timeout":     "not-a-duration",
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want decode failure")
	}
}

func TestFactory_CreateMetrics_MissingMetricsPrefixes(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"project_ids": []string{"my-project"},
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want validation failure")
	}
}

func TestFactory_CreateMetrics_UnknownField(t *testing.T) {
	t.Parallel()

	err := createMetrics(t, map[string]interface{}{
		"metrics_prefixes": []string{"compute.googleapis.com/instance"},
		"not_a_real_field": "value",
	})
	if err == nil {
		t.Fatal("CreateMetrics() error = nil, want unknown-field failure")
	}
}
