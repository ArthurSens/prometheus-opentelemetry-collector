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
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/mitchellh/mapstructure"
	"github.com/prometheus-community/stackdriver_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
)

var _ prombridge.ConfigDecoder = configDecoder{}

type configDecoder struct{}

func (configDecoder) DecodeConfig(raw map[string]interface{}) (any, error) {
	cfg := defaultConfig()
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:     cfg,
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		MatchName: func(yamlKey, goField string) bool {
			return strings.EqualFold(strings.ReplaceAll(yamlKey, "_", ""), goField)
		},
		ErrorUnused: true,
	})
	if err != nil {
		return nil, err
	}
	if err := dec.Decode(raw); err != nil {
		return nil, fmt.Errorf("decode receiver config: %w", err)
	}
	return cfg, nil
}

func defaultConfig() *config.Config {
	return &config.Config{
		UniverseDomain:            config.DefaultUniverseDomain,
		MaxRetries:                config.DefaultMaxRetries,
		HTTPTimeout:               config.DefaultHTTPTimeout,
		MaxBackoff:                config.DefaultMaxBackoff,
		BackoffJitter:             config.DefaultBackoffJitter,
		RetryStatuses:             slices.Clone(config.DefaultRetryStatuses),
		MetricsInterval:           config.DefaultMetricsInterval,
		MetricsOffset:             config.DefaultMetricsOffset,
		MetricsIngestDelay:        config.DefaultMetricsIngest,
		FillMissingLabels:         config.DefaultFillMissing,
		DropDelegatedProjects:     config.DefaultDropDelegated,
		AggregateDeltas:           config.DefaultAggregateDeltas,
		AggregateDeltasTTL:        config.DefaultDeltasTTL,
		DescriptorCacheTTL:        config.DefaultDescriptorTTL,
		DescriptorCacheOnlyGoogle: config.DefaultDescriptorGoogleOnly,
	}
}

// componentDefaults generates the OTel-framework defaults map by walking the typed
// default Config. time.Duration fields are emitted as their .String() form to match
// the YAML representation; everything else passes through.
func componentDefaults() map[string]interface{} {
	out := map[string]interface{}{}
	rv := reflect.ValueOf(defaultConfig()).Elem()
	rt := rv.Type()
	for i := range rt.NumField() {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		key := snakeCase(f.Name)
		v := rv.Field(i).Interface()
		if d, ok := v.(time.Duration); ok {
			out[key] = d.String()
			continue
		}
		out[key] = v
	}
	return out
}

// snakeCase converts a PascalCase or camelCase Go identifier to snake_case,
// handling acronyms (HTTPTimeout → http_timeout, ProjectIDs → project_ids,
// AggregateDeltasTTL → aggregate_deltas_ttl).
func snakeCase(name string) string {
	runes := []rune(name)
	var out []rune
	for i, r := range runes {
		if i == 0 {
			out = append(out, unicode.ToLower(r))
			continue
		}
		prev := runes[i-1]
		if unicode.IsUpper(r) {
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				// lowercase→uppercase boundary: always split.
				out = append(out, '_')
			} else if unicode.IsUpper(prev) {
				// Within an uppercase run: split before this letter only if the
				// run has 3+ uppercase letters (i.e., two-back is also uppercase).
				// This splits "HTTP"+"Timeout" but keeps "ID"+"s" together.
				next := rune(0)
				if i+1 < len(runes) {
					next = runes[i+1]
				}
				if next != 0 && unicode.IsLower(next) {
					prevPrev := rune(0)
					if i >= 2 {
						prevPrev = runes[i-2]
					}
					if unicode.IsUpper(prevPrev) {
						out = append(out, '_')
					}
				}
			}
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}
