.PHONY: generate build test check clean

OCB_VERSION ?= v0.151.0
MANIFEST := builder-config.yaml
BUILD_DIR := _build
BUILDER_BIN := $(CURDIR)/.bin/builder
RECEIVER_MODULES := receivers/stackdriver

$(BUILDER_BIN):
	mkdir -p "$(dir $(BUILDER_BIN))"
	GOBIN="$(CURDIR)/.bin" go install go.opentelemetry.io/collector/cmd/builder@$(OCB_VERSION)

generate: $(BUILDER_BIN)
	"$(BUILDER_BIN)" --skip-compilation --config "$(MANIFEST)"

build: $(BUILDER_BIN)
	"$(BUILDER_BIN)" --config "$(MANIFEST)"

test:
	@for module in $(RECEIVER_MODULES); do \
		echo "==> go test ./... in $$module"; \
		(cd "$$module" && go test ./...); \
	done

check: generate
	cd "$(BUILD_DIR)" && go build ./...

clean:
	rm -rf "$(BUILD_DIR)" "$(CURDIR)/.bin"
