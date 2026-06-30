.PHONY: generate-distribution gogenerate mdatagen check-metadata build test check chlog-new chlog-validate chlog-preview chlog-update clean

GOCMD ?= go
OCB_VERSION ?= v0.155.0
SRC_ROOT := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
MANIFEST := $(SRC_ROOT)/builder-config.yaml
BUILD_DIR := $(SRC_ROOT)/_build
BUILDER_BIN := $(SRC_ROOT)/.bin/builder
TOOLS_MOD_DIR := $(SRC_ROOT)/internal/tools
GO_TOOL := GOOS= GOARCH= $(GOCMD) -C $(TOOLS_MOD_DIR) tool
CHLOGGEN := $(subst \,/,$(shell $(GO_TOOL) -n go.opentelemetry.io/build-tools/chloggen))
MDATAGEN := $(subst \,/,$(shell $(GO_TOOL) -n go.opentelemetry.io/collector/cmd/mdatagen))
RECEIVER_MODULES := $(patsubst $(SRC_ROOT)/%/go.mod,%,$(wildcard $(SRC_ROOT)/receivers/*/go.mod))
CHLOGGEN_CONFIG := .chloggen/config.yaml

$(BUILDER_BIN):
	mkdir -p "$(dir $(BUILDER_BIN))"
	GOBIN="$(SRC_ROOT)/.bin" $(GOCMD) install go.opentelemetry.io/collector/cmd/builder@$(OCB_VERSION)

generate-distribution: $(BUILDER_BIN)
	"$(BUILDER_BIN)" --skip-compilation --config "$(MANIFEST)"

MDATAGEN_METADATA_YAML ?= metadata.yaml

mdatagen:
	@$(MDATAGEN) $(MDATAGEN_METADATA_YAML)

gogenerate:
	@for module in $(RECEIVER_MODULES); do \
		echo "==> go generate ./... in $$module"; \
		(cd "$(SRC_ROOT)/$$module" && $(GOCMD) generate ./...); \
	done

check-metadata: gogenerate
	cd "$(SRC_ROOT)" && git diff --exit-code
	cd "$(SRC_ROOT)" && test -z "$$(git status --porcelain)" # catches new file creations

build: $(BUILDER_BIN)
	"$(BUILDER_BIN)" --config "$(MANIFEST)"

test:
	@for module in $(RECEIVER_MODULES); do \
		echo "==> go test ./... in $$module"; \
		(cd "$(SRC_ROOT)/$$module" && $(GOCMD) test ./...); \
	done

check: generate-distribution
	cd "$(BUILD_DIR)" && go build ./...

chlog-new:
	@test -n "$(NAME)" || (echo "NAME is required. Usage: make chlog-new NAME=<entry-name>" && exit 1)
	cd "$(SRC_ROOT)" && "$(CHLOGGEN)" --config "$(CHLOGGEN_CONFIG)" new -filename "$(NAME)"

chlog-validate:
	cd "$(SRC_ROOT)" && "$(CHLOGGEN)" --config "$(CHLOGGEN_CONFIG)" validate

chlog-preview:
	cd "$(SRC_ROOT)" && "$(CHLOGGEN)" --config "$(CHLOGGEN_CONFIG)" update --dry

chlog-update:
	@test -n "$(VERSION)" || (echo "VERSION is required. Usage: make chlog-update VERSION=v0.1.0" && exit 1)
	cd "$(SRC_ROOT)" && "$(CHLOGGEN)" --config "$(CHLOGGEN_CONFIG)" update --version "$(VERSION)"

clean:
	rm -rf "$(BUILD_DIR)" "$(SRC_ROOT)/.bin"
