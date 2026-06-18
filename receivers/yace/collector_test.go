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

package yace

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/promutil"
	"github.com/prometheus/client_golang/prometheus"
)

type fakeScraper struct {
	metrics []*promutil.PrometheusMetric
	err     error
	calls   int
}

func (s *fakeScraper) Scrape(context.Context) ([]*promutil.PrometheusMetric, error) {
	s.calls++
	return s.metrics, s.err
}

func TestScrapeCollector_CollectsGeneratedMetrics(t *testing.T) {
	t.Parallel()

	scraper := &fakeScraper{
		metrics: []*promutil.PrometheusMetric{
			{
				Name:   "aws_test_metric",
				Labels: map[string]string{"region": "us-east-1"},
				Value:  42,
			},
		},
	}
	registry := prometheus.NewRegistry()
	if err := registry.Register(newScrapeCollector(context.Background(), discardLogger(), scraper)); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	if scraper.calls != 1 {
		t.Fatalf("Scrape() calls = %d, want 1", scraper.calls)
	}

	for _, family := range families {
		if family.GetName() != "aws_test_metric" {
			continue
		}
		if got := family.GetMetric()[0].GetGauge().GetValue(); got != 42 {
			t.Fatalf("aws_test_metric value = %v, want 42", got)
		}
		return
	}
	t.Fatal("Gather() did not return aws_test_metric")
}

func TestScrapeCollector_ReportsScrapeError(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	scraper := &fakeScraper{err: errors.New("scrape failed")}
	registry := prometheus.NewRegistry()
	if err := registry.Register(newScrapeCollector(context.Background(), testLogger(&logs), scraper)); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}
	if scraper.calls != 1 {
		t.Fatalf("Scrape() calls = %d, want 1", scraper.calls)
	}
	if got := logs.String(); !strings.Contains(got, "YACE scrape failed") || !strings.Contains(got, "scrape failed") {
		t.Fatalf("logged output = %q, want scrape error", got)
	}
}

func discardLogger() *slog.Logger {
	return testLogger(io.Discard)
}

func testLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, nil))
}
