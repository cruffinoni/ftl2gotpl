.DEFAULT_GOAL := help

GO ?= go
BIN_DIR ?= bin
BIN ?= $(BIN_DIR)/ftl2gotpl

IN ?= ../templates_download
OUT ?= ./out
SAMPLES ?= ./testdata/samples
ARTIFACTS ?= ./artifacts
STRICT ?= false
EXT ?= .gotmpl

.PHONY: help fmt test build run convert render-check clean

help: ## Show available targets and variables
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Variables (override like: make convert IN=../templates_download):"
	@echo "  IN=$(IN)"
	@echo "  OUT=$(OUT)"
	@echo "  SAMPLES=$(SAMPLES)"
	@echo "  ARTIFACTS=$(ARTIFACTS)"
	@echo "  STRICT=$(STRICT)"
	@echo "  EXT=$(EXT)"

fmt: ## Format Go source files
	$(GO) fmt ./...

test: ## Run all tests
	$(GO) test ./...

build: ## Build binary to $(BIN)
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN) ./cmd/ftl2gotpl

run: ## Run converter with default flags
	$(GO) run ./cmd/ftl2gotpl --in $(IN) --out $(OUT) --ext $(EXT) --strict=$(STRICT)

convert: ## Convert templates and write JSON/CSV reports
	mkdir -p $(ARTIFACTS)
	$(GO) run ./cmd/ftl2gotpl \
		--in $(IN) \
		--out $(OUT) \
		--ext $(EXT) \
		--strict=$(STRICT) \
		--report-json $(ARTIFACTS)/report.json \
		--report-csv $(ARTIFACTS)/report.csv

render-check: ## Convert with render check enabled
	mkdir -p $(ARTIFACTS)
	$(GO) run ./cmd/ftl2gotpl \
		--in $(IN) \
		--out $(OUT) \
		--ext $(EXT) \
		--strict=$(STRICT) \
		--render-check \
		--samples-root $(SAMPLES) \
		--report-json $(ARTIFACTS)/report.json \
		--report-csv $(ARTIFACTS)/report.csv

clean: ## Remove generated outputs
	rm -rf $(BIN_DIR) $(OUT) $(ARTIFACTS)
