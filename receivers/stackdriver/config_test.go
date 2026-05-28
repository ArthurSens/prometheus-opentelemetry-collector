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

package otelcollector

import (
	"maps"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/prometheus-community/stackdriver_exporter/config"
	"github.com/stretchr/testify/require"
)

func TestSnakeCase(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input string
		want  string
	}{
		{"ProjectIDs", "project_ids"},
		{"HTTPTimeout", "http_timeout"},
		{"MaxRetries", "max_retries"},
		{"AggregateDeltasTTL", "aggregate_deltas_ttl"},
		{"DescriptorCacheOnlyGoogle", "descriptor_cache_only_google"},
		{"MetricsIngestDelay", "metrics_ingest_delay"},
		{"ProjectsFilter", "projects_filter"},
		{"UniverseDomain", "universe_domain"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := snakeCase(tc.input)
			if got != tc.want {
				t.Fatalf("snakeCase(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestComponentDefaultsKeysMirrorConfig(t *testing.T) {
	t.Parallel()

	want := []string{
		"project_ids", "projects_filter", "universe_domain",
		"max_retries", "http_timeout", "max_backoff", "backoff_jitter",
		"retry_statuses", "metrics_prefixes", "metrics_interval",
		"metrics_offset", "metrics_ingest_delay", "fill_missing_labels",
		"drop_delegated_projects", "filters", "aggregate_deltas",
		"aggregate_deltas_ttl", "descriptor_cache_ttl", "descriptor_cache_only_google",
	}
	got := slices.Sorted(maps.Keys(componentDefaults()))
	slices.Sort(want)
	require.Equal(t, want, got)
}

func TestDecodeConfig_DefaultsMatchDefaultConfig(t *testing.T) {
	t.Parallel()

	want := defaultConfig()

	got, err := configDecoder{}.DecodeConfig(componentDefaults())
	if err != nil {
		t.Fatalf("DecodeConfig(componentDefaults()) error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DecodeConfig(componentDefaults()) = %#v, want %#v", got, want)
	}
}

func TestDecodeConfig_FullRoundTrip(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"project_ids":                  []string{"proj-a", "proj-b"},
		"projects_filter":              "parent.type:folder parent.id:123",
		"universe_domain":              "example.com",
		"max_retries":                  3,
		"http_timeout":                 "30s",
		"max_backoff":                  "10s",
		"backoff_jitter":               "2s",
		"retry_statuses":               []int{429, 503},
		"metrics_prefixes":             []string{"compute.googleapis.com/instance"},
		"metrics_interval":             "10m",
		"metrics_offset":               "1m",
		"metrics_ingest_delay":         true,
		"fill_missing_labels":          false,
		"drop_delegated_projects":      true,
		"filters":                      []string{"compute.googleapis.com/instance:resource.labels.zone=us-central1-a"},
		"aggregate_deltas":             true,
		"aggregate_deltas_ttl":         "1h",
		"descriptor_cache_ttl":         "15m",
		"descriptor_cache_only_google": false,
	}

	result, err := configDecoder{}.DecodeConfig(raw)
	if err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}

	cfg, ok := result.(*config.Config)
	if !ok {
		t.Fatalf("DecodeConfig() returned unexpected type %T", result)
	}

	if !reflect.DeepEqual(cfg.ProjectIDs, []string{"proj-a", "proj-b"}) {
		t.Errorf("ProjectIDs = %v, want [proj-a proj-b]", cfg.ProjectIDs)
	}
	if cfg.ProjectsFilter != "parent.type:folder parent.id:123" {
		t.Errorf("ProjectsFilter = %q", cfg.ProjectsFilter)
	}
	if cfg.UniverseDomain != "example.com" {
		t.Errorf("UniverseDomain = %q", cfg.UniverseDomain)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.HTTPTimeout != 30*time.Second {
		t.Errorf("HTTPTimeout = %v, want 30s", cfg.HTTPTimeout)
	}
	if cfg.MaxBackoff != 10*time.Second {
		t.Errorf("MaxBackoff = %v, want 10s", cfg.MaxBackoff)
	}
	if cfg.BackoffJitter != 2*time.Second {
		t.Errorf("BackoffJitter = %v, want 2s", cfg.BackoffJitter)
	}
	if !reflect.DeepEqual(cfg.RetryStatuses, []int{429, 503}) {
		t.Errorf("RetryStatuses = %v, want [429 503]", cfg.RetryStatuses)
	}
	if !reflect.DeepEqual(cfg.MetricsPrefixes, []string{"compute.googleapis.com/instance"}) {
		t.Errorf("MetricsPrefixes = %v", cfg.MetricsPrefixes)
	}
	if !reflect.DeepEqual(cfg.Filters, []string{"compute.googleapis.com/instance:resource.labels.zone=us-central1-a"}) {
		t.Errorf("Filters = %v", cfg.Filters)
	}
	if cfg.MetricsInterval != 10*time.Minute {
		t.Errorf("MetricsInterval = %v, want 10m", cfg.MetricsInterval)
	}
	if cfg.MetricsOffset != 1*time.Minute {
		t.Errorf("MetricsOffset = %v, want 1m", cfg.MetricsOffset)
	}
	if !cfg.MetricsIngestDelay {
		t.Errorf("MetricsIngestDelay = false, want true")
	}
	if cfg.FillMissingLabels {
		t.Errorf("FillMissingLabels = true, want false")
	}
	if !cfg.DropDelegatedProjects {
		t.Errorf("DropDelegatedProjects = false, want true")
	}
	if !cfg.AggregateDeltas {
		t.Errorf("AggregateDeltas = false, want true")
	}
	if cfg.AggregateDeltasTTL != 1*time.Hour {
		t.Errorf("AggregateDeltasTTL = %v, want 1h", cfg.AggregateDeltasTTL)
	}
	if cfg.DescriptorCacheTTL != 15*time.Minute {
		t.Errorf("DescriptorCacheTTL = %v, want 15m", cfg.DescriptorCacheTTL)
	}
	if cfg.DescriptorCacheOnlyGoogle {
		t.Errorf("DescriptorCacheOnlyGoogle = true, want false")
	}
}

func TestDecodeConfig_DurationStringOverridesDefault(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"metrics_prefixes": []string{"compute.googleapis.com/instance"},
		"http_timeout":     "45s",
	}

	result, err := configDecoder{}.DecodeConfig(raw)
	if err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}

	cfg := result.(*config.Config)
	if cfg.HTTPTimeout != 45*time.Second {
		t.Fatalf("HTTPTimeout = %v, want 45s", cfg.HTTPTimeout)
	}
	if cfg.FillMissingLabels != config.DefaultFillMissing {
		t.Fatalf("FillMissingLabels = %v, want default %v", cfg.FillMissingLabels, config.DefaultFillMissing)
	}
	if cfg.MaxBackoff != config.DefaultMaxBackoff {
		t.Fatalf("MaxBackoff = %v, want default %v", cfg.MaxBackoff, config.DefaultMaxBackoff)
	}
}

func TestDecodeConfig_UnknownKey(t *testing.T) {
	t.Parallel()

	raw := map[string]interface{}{
		"unknown_key": "value",
	}

	_, err := configDecoder{}.DecodeConfig(raw)
	if err == nil {
		t.Fatal("DecodeConfig() expected error for unknown key, got nil")
	}
}
