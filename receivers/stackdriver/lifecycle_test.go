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
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/prometheus-community/stackdriver_exporter/collectors"
	"github.com/prometheus-community/stackdriver_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/api/monitoring/v3"
)

func TestLifecycleManager_Start(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		ProjectIDs:      []string{"project-a", "project-b"},
		MetricsPrefixes: []string{"compute.googleapis.com/instance"},
		HTTPTimeout:     "10s",
		MaxBackoff:      "5s",
		BackoffJitter:   "1s",
		MetricsInterval: "5m",
		MetricsOffset:   "0s",
		DeltasTTL:       "30m",
		DescriptorTTL:   "0s",
	}

	var createdProjects []string
	var gotOpts collectors.MonitoringCollectorOptions
	var gotDeltasTTL time.Duration
	mgr := newLifecycleManager()
	mgr.monitoringServiceFactory = func(context.Context, config.RuntimeConfig) (*monitoring.Service, error) {
		return &monitoring.Service{}, nil
	}
	mgr.collectorFactory = func(projectID string, _ *monitoring.Service, opts collectors.MonitoringCollectorOptions, deltasTTL time.Duration, _ *slog.Logger) (prometheus.Collector, error) {
		createdProjects = append(createdProjects, projectID)
		gotOpts = opts
		gotDeltasTTL = deltasTTL
		return prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "test_metric_" + strings.ReplaceAll(projectID, "-", "_"),
			Help: "test",
		}), nil
	}

	reg, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if reg == nil {
		t.Fatal("Start() returned nil registry")
	}
	if len(createdProjects) != 2 {
		t.Fatalf("collectorFactory called %d times, want 2", len(createdProjects))
	}
	wantOpts := collectors.MonitoringCollectorOptions{
		MetricTypePrefixes:        config.ParseMetricPrefixes(cfg.MetricsPrefixes),
		ExtraFilters:              config.ParseMetricFilters(cfg.Filters),
		RequestInterval:           5 * time.Minute,
		RequestOffset:             0,
		IngestDelay:               false,
		FillMissingLabels:         false,
		DropDelegatedProjects:     false,
		AggregateDeltas:           false,
		DescriptorCacheTTL:        0,
		DescriptorCacheOnlyGoogle: false,
	}
	if !reflect.DeepEqual(gotOpts, wantOpts) {
		t.Fatalf("collector options = %#v, want %#v", gotOpts, wantOpts)
	}
	if gotDeltasTTL != 30*time.Minute {
		t.Fatalf("deltas TTL = %v, want %v", gotDeltasTTL, 30*time.Minute)
	}

	if _, err := reg.Gather(); err != nil {
		t.Fatalf("registry.Gather() error = %v", err)
	}
}

func TestLifecycleManager_Start_UsesCollectorLogger(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		ProjectIDs:      []string{"project-a"},
		MetricsPrefixes: []string{"compute.googleapis.com/instance"},
		HTTPTimeout:     "10s",
		MaxBackoff:      "5s",
		BackoffJitter:   "1s",
		MetricsInterval: "5m",
		MetricsOffset:   "0s",
		DeltasTTL:       "30m",
		DescriptorTTL:   "0s",
	}

	core, observed := observer.New(zap.DebugLevel)
	settings := receivertest.NewNopSettings(receiverType)
	settings.Logger = zap.New(core)

	mgr := newLifecycleManager()
	mgr.monitoringServiceFactory = func(context.Context, config.RuntimeConfig) (*monitoring.Service, error) {
		return &monitoring.Service{}, nil
	}
	mgr.collectorFactory = func(projectID string, _ *monitoring.Service, _ collectors.MonitoringCollectorOptions, _ time.Duration, logger *slog.Logger) (prometheus.Collector, error) {
		logger.Info("using collector logger", "project_id", projectID)
		return prometheus.NewGauge(prometheus.GaugeOpts{Name: "collector_logger_metric", Help: "test"}), nil
	}

	if _, err := mgr.Start(context.Background(), settings, cfg); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	entries := observed.FilterMessage("using collector logger").AllUntimed()
	if len(entries) != 1 {
		t.Fatalf("logged entries = %d, want 1", len(entries))
	}
	if got := entries[0].ContextMap()["project_id"]; got != "project-a" {
		t.Fatalf("logged project_id = %#v, want %q", got, "project-a")
	}
}

func TestLifecycleManager_Start_UsesDefaultProjectDiscovery(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		MetricsPrefixes: []string{"compute.googleapis.com/instance"},
		HTTPTimeout:     "10s",
		MaxBackoff:      "5s",
		BackoffJitter:   "1s",
		MetricsInterval: "5m",
		MetricsOffset:   "0s",
		DeltasTTL:       "30m",
		DescriptorTTL:   "0s",
	}

	mgr := newLifecycleManager()
	mgr.defaultProjectDiscoverer = func(context.Context) (string, error) {
		return "auto-project", nil
	}
	mgr.monitoringServiceFactory = func(context.Context, config.RuntimeConfig) (*monitoring.Service, error) {
		return &monitoring.Service{}, nil
	}
	mgr.collectorFactory = func(projectID string, _ *monitoring.Service, _ collectors.MonitoringCollectorOptions, _ time.Duration, _ *slog.Logger) (prometheus.Collector, error) {
		if projectID != "auto-project" {
			t.Fatalf("projectID = %q, want %q", projectID, "auto-project")
		}
		return prometheus.NewGauge(prometheus.GaugeOpts{Name: "auto_project_metric", Help: "test"}), nil
	}

	if _, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), cfg); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

func TestLifecycleManager_StartsAndShutsDownMultipleInstancesIndependently(t *testing.T) {
	t.Parallel()

	type instance struct {
		name                  string
		projectID             string
		metricName            string
		monitoringCreateCalls int
		collectorCreateCalls  int
		mgr                   *lifecycleManager
		cfg                   *Config
		registry              *prometheus.Registry
	}

	instances := []*instance{
		{
			name:       "first",
			projectID:  "project-a",
			metricName: "first_stackdriver_instance_metric",
		},
		{
			name:       "second",
			projectID:  "project-b",
			metricName: "second_stackdriver_instance_metric",
		},
	}

	for _, inst := range instances {
		inst.cfg = &Config{
			ProjectIDs:      []string{inst.projectID},
			MetricsPrefixes: []string{"compute.googleapis.com/instance"},
			HTTPTimeout:     "10s",
			MaxBackoff:      "5s",
			BackoffJitter:   "1s",
			MetricsInterval: "5m",
			MetricsOffset:   "0s",
			DeltasTTL:       "30m",
			DescriptorTTL:   "0s",
		}

		inst.mgr = newLifecycleManager()
		inst.mgr.monitoringServiceFactory = func(context.Context, config.RuntimeConfig) (*monitoring.Service, error) {
			inst.monitoringCreateCalls++
			return &monitoring.Service{}, nil
		}
		inst.mgr.collectorFactory = func(projectID string, _ *monitoring.Service, _ collectors.MonitoringCollectorOptions, _ time.Duration, _ *slog.Logger) (prometheus.Collector, error) {
			inst.collectorCreateCalls++
			if projectID != inst.projectID {
				t.Fatalf("%s projectID = %q, want %q", inst.name, projectID, inst.projectID)
			}
			return prometheus.NewGauge(prometheus.GaugeOpts{Name: inst.metricName, Help: "test"}), nil
		}
	}

	for _, inst := range instances {
		registry, err := inst.mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), inst.cfg)
		if err != nil {
			t.Fatalf("%s Start() error = %v", inst.name, err)
		}
		if registry == nil {
			t.Fatalf("%s Start() returned nil registry", inst.name)
		}
		inst.registry = registry
	}

	for _, inst := range instances {
		if inst.monitoringCreateCalls != 1 {
			t.Fatalf("%s monitoringCreateCalls = %d, want 1", inst.name, inst.monitoringCreateCalls)
		}
		if inst.collectorCreateCalls != 1 {
			t.Fatalf("%s collectorCreateCalls = %d, want 1", inst.name, inst.collectorCreateCalls)
		}
		if !registryContainsMetric(t, inst.registry, inst.metricName) {
			t.Fatalf("%s registry does not contain metric %q", inst.name, inst.metricName)
		}
	}

	if registryContainsMetric(t, instances[0].registry, instances[1].metricName) {
		t.Fatalf("%s registry unexpectedly contains metric %q", instances[0].name, instances[1].metricName)
	}
	if registryContainsMetric(t, instances[1].registry, instances[0].metricName) {
		t.Fatalf("%s registry unexpectedly contains metric %q", instances[1].name, instances[0].metricName)
	}

	if err := instances[0].mgr.Shutdown(context.Background()); err != nil {
		t.Fatalf("%s Shutdown() error = %v", instances[0].name, err)
	}
	if !registryContainsMetric(t, instances[1].registry, instances[1].metricName) {
		t.Fatalf("%s registry lost metric %q after shutting down %s", instances[1].name, instances[1].metricName, instances[0].name)
	}
	if err := instances[1].mgr.Shutdown(context.Background()); err != nil {
		t.Fatalf("%s Shutdown() error = %v", instances[1].name, err)
	}
}

func TestLifecycleManager_Start_ReturnsErrorFromCollectorFactory(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		ProjectIDs:      []string{"project-a"},
		MetricsPrefixes: []string{"compute.googleapis.com/instance"},
		HTTPTimeout:     "10s",
		MaxBackoff:      "5s",
		BackoffJitter:   "1s",
		MetricsInterval: "5m",
		MetricsOffset:   "0s",
		DeltasTTL:       "30m",
		DescriptorTTL:   "0s",
	}

	mgr := newLifecycleManager()
	mgr.monitoringServiceFactory = func(context.Context, config.RuntimeConfig) (*monitoring.Service, error) {
		return &monitoring.Service{}, nil
	}
	mgr.collectorFactory = func(string, *monitoring.Service, collectors.MonitoringCollectorOptions, time.Duration, *slog.Logger) (prometheus.Collector, error) {
		return nil, errors.New("boom")
	}

	if _, err := mgr.Start(context.Background(), receivertest.NewNopSettings(receiverType), cfg); err == nil {
		t.Fatal("Start() expected error, got nil")
	}
}

func TestLifecycleManager_Shutdown(t *testing.T) {
	t.Parallel()

	mgr := newLifecycleManager()
	if err := mgr.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func registryContainsMetric(t *testing.T, registry *prometheus.Registry, metricName string) bool {
	t.Helper()

	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("registry.Gather() error = %v", err)
	}

	for _, metricFamily := range metricFamilies {
		if metricFamily.GetName() == metricName {
			return true
		}
	}
	return false
}
