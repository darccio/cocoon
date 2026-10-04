SHELL := /bin/bash
export GOCACHE := $(CURDIR)/.cache/go
export GOLANGCI_LINT_CACHE := $(CURDIR)/.cache/lint
export GOFLAGS := -buildvcs=false

.PHONY: check test lint fmt race coverage cross rust
check: test lint
test:
	CGO_ENABLED=0 go test ./...
	CGO_ENABLED=0 go vet ./...
lint:
	golangci-lint fmt --diff
	golangci-lint run
fmt:
	golangci-lint fmt
race:
	CGO_ENABLED=1 go test -race ./...
coverage:
	CGO_ENABLED=0 go test -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
cross:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./...
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build ./...
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...
	CGO_ENABLED=0 GOOS=js GOARCH=wasm go build ./...
rust:
	cargo fmt --all --check
	cargo clippy --workspace --all-targets -- -D warnings
	cargo test --workspace --locked
