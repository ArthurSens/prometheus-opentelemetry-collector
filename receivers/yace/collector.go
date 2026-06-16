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
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/prometheus-community/yet-another-cloudwatch-exporter/pkg/promutil"
	"github.com/prometheus/client_golang/prometheus"
)

// scraper is the minimal YACE behavior this collector needs; tests use it to
// inject a fake without constructing AWS clients.
type scraper interface {
	Scrape(context.Context) ([]*promutil.PrometheusMetric, error)
}

type scrapeCollector struct {
	ctx     context.Context
	logger  *slog.Logger
	scraper scraper
	mu      sync.Mutex
}

func newScrapeCollector(ctx context.Context, logger *slog.Logger, scraper scraper) *scrapeCollector {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &scrapeCollector{
		ctx:     ctx,
		logger:  logger,
		scraper: scraper,
	}
}

func (c *scrapeCollector) Describe(chan<- *prometheus.Desc) {
	// YACE returns a dynamic metric set, so this collector is intentionally
	// unchecked and emits descriptors only after each CloudWatch scrape.
}

func (c *scrapeCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	generatedMetrics, err := c.scraper.Scrape(c.ctx)
	if err != nil {
		c.logger.Error("YACE scrape failed", "err", err)
		return
	}

	promutil.NewPrometheusCollector(generatedMetrics).Collect(ch)
}
