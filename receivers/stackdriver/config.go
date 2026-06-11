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
	"github.com/prometheus-community/stackdriver_exporter/config"
	prombridge "github.com/prometheus/opentelemetry-collector-bridge"
)

var _ prombridge.ConfigUnmarshaler = configUnmarshaler{}

// configUnmarshaler hands the bridge a defaulted stackdriver config.Config to
// decode the receiver's settings into. The bridge's untagged decoder maps the
// snake_case OTel keys onto the upstream struct's fields, derives the component
// defaults from the same struct, and validates it via config.Config.Validate;
// the duration decode hook is wired in NewFactory.
type configUnmarshaler struct{}

func (configUnmarshaler) GetConfigStruct() any {
	return config.NewConfigWithDefaults()
}
