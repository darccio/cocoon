SHELL := /bin/bash
export GOCACHE := $(CURDIR)/.cache/go
export GOLANGCI_LINT_CACHE := $(CURDIR)/.cache/lint
export GOFLAGS := -buildvcs=false
BINARYEN_BIN ?= $(CURDIR)/.cache/binaryen/binaryen-version_133/bin
export PATH := $(BINARYEN_BIN):$(PATH)
GOLANGCI_LINT ?= $(if $(wildcard $(CURDIR)/.cache/bin/golangci-lint),$(CURDIR)/.cache/bin/golangci-lint,golangci-lint)
PACKAGES := ./... ./testdata/compute/go/compute/...

.PHONY: check test lint fmt race coverage cross rust integration smoke fuzz bench licenses licenses-check
check: test lint

# Resolve every Go module and Cargo workspace, including indirect/test/tool deps.
licenses:
	go run -mod=readonly ./tools/licenses

licenses-check:
	go run -mod=readonly ./tools/licenses -check
test:
	CGO_ENABLED=0 go test $(PACKAGES)
	CGO_ENABLED=0 go vet $(PACKAGES)
lint:
	$(GOLANGCI_LINT) fmt --diff $(PACKAGES)
	$(GOLANGCI_LINT) run $(PACKAGES)
fmt:
	$(GOLANGCI_LINT) fmt $(PACKAGES)
race:
	CGO_ENABLED=1 go test -race $(PACKAGES)
coverage:
	CGO_ENABLED=0 go test -covermode=atomic -coverprofile=coverage.out $(PACKAGES)
	go tool cover -func=coverage.out
cross:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go test -exec=true $(PACKAGES)
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go test -exec=true $(PACKAGES)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go test -exec=true $(PACKAGES)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go test -exec=true $(PACKAGES)
	CGO_ENABLED=0 GOOS=js GOARCH=wasm go test -exec=true $(PACKAGES)
rust:
	cargo fmt --all --check
	cargo clippy --workspace --all-features --all-targets -- -D warnings
	cargo test --workspace --all-features --locked
integration:
	go run ./cmd/cocoon doctor
	go run ./cmd/cocoon build --manifest testdata/compute/cocoon.toml
	go run ./cmd/cocoon build --manifest examples/datadog/cocoon.toml
	go run ./cmd/cocoon verify --manifest testdata/compute/cocoon.toml testdata/compute/go/compute/testdata/module.wasm
	go run ./cmd/cocoon verify --manifest examples/datadog/cocoon.toml examples/datadog/go/dd/testdata/module.wasm
	$(MAKE) test rust
	cargo fmt --manifest-path testdata/compute/shim/Cargo.toml --check
	cargo clippy --manifest-path testdata/compute/shim/Cargo.toml --all-targets -- -D warnings
	cargo test --manifest-path testdata/compute/shim/Cargo.toml --locked
	cargo fmt --manifest-path examples/datadog/shim/Cargo.toml --check
	cargo clippy --manifest-path examples/datadog/shim/Cargo.toml --all-targets -- -D warnings
	cargo test --manifest-path examples/datadog/shim/Cargo.toml --locked
smoke:
	COCOON_SMOKE=1 CGO_ENABLED=0 go test ./internal/smoke -run '^Test(ExternalWorkflow|OptimizerSemantics|NumericLoweringSemantics)$$' -count=1 -timeout=10m -v
fuzz:
	go test ./internal/manifest -run '^$$' -fuzz FuzzParse -fuzztime=15s -parallel=2
	go test ./internal/wasmbin -run '^$$' -fuzz FuzzRead -fuzztime=15s -parallel=2
	go test ./internal/harden -run '^$$' -fuzz FuzzRewrite -fuzztime=15s -parallel=2
	go test ./testdata/compute/go/compute -run '^$$' -fuzz FuzzGeneratedDifferential -fuzztime=15s -parallel=2
	go test ./examples/datadog/go/dd -run '^$$' -fuzz FuzzGeneratedDifferential -fuzztime=15s -parallel=2
bench:
	CGO_ENABLED=0 go test ./examples/datadog/go/dd -run '^$$' -bench . -benchmem -count=5
