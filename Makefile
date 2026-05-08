SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=
PROTOC_FAILURE_MESSAGE="\nFailed to compile protobuf files: protoc exited with code $$?. Ensure you have the latest version of protoc: https://grpc.io/docs/protoc-installation/\n"

export GO111MODULE := on
export GOBIN := $(shell pwd)/bin
export PATH := $(GOBIN):$(PATH)
export GOLANGCI_LINT_VERSION := v2.10.1

# Install all the build and lint dependencies
setup:
	go mod download
.PHONY: setup

# Initialize the pre-commit git hook
githooks-init:
	cp .pre-commit .git/hooks/pre-commit
.PHONY: githooks-init

# Run all the tests
# On macOS, CGO_ENABLED=0 works around a Go 1.26.0 linker crash with -race (https://github.com/golang/go/issues/77593)
TEST_CGO_ENABLED := $(if $(filter Darwin,$(shell uname -s)),0,)
test:
	$(if $(TEST_CGO_ENABLED),CGO_ENABLED=$(TEST_CGO_ENABLED)) go test $(TEST_OPTIONS) -failfast -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=2m
.PHONY: test

# Run all the tests and opens the coverage report
cover: test
	go tool cover -html=coverage.txt
.PHONY: cover

# Build a production-like binary for canary tests
build-canary:
	go generate ./...
	CGO_ENABLED=0 go build -ldflags="-s -w" -o stripe ./cmd/stripe
.PHONY: build-canary

# Run canary tests (end-to-end tests against the compiled binary)
canary: build-canary
	STRIPE_CLI_BINARY=$(PWD)/stripe go test -v -timeout 15m ./canary/...
.PHONY: canary

# Run offline canary tests only (no API key required)
canary-offline: build-canary
	STRIPE_CLI_BINARY=$(PWD)/stripe go test -v -timeout 10m ./canary/... -run "TestOffline"
.PHONY: canary-offline

# gofmt and goimports all go files
fmt:
	go install golang.org/x/tools/cmd/goimports@v0.42.0
	find . -not -path "./rpc*" -not -path "./pkg/plugins/proto*" -name '*.go' | while read -r file; do gofmt -w -s "$$file"; goimports -w -local github.com/stripe/stripe-cli "$$file"; done
.PHONY: fmt

# Run all the linters
lint: bin/golangci-lint
	./bin/golangci-lint run --max-issues-per-linter=0 --max-same-issues=0 ./...
.PHONY: lint

bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s $(GOLANGCI_LINT_VERSION)

# Clean go.mod
go-mod-tidy:
	@go mod tidy -v
	@git diff HEAD
	@git diff-index --quiet HEAD
.PHONY: go-mod-tidy

# Run all the tests and code checks
ci: build-all-platforms test go-mod-tidy protoc-ci
.PHONY: ci

# Build a beta version of stripe
build:
	go generate ./...
	go build -o stripe cmd/stripe/main.go
.PHONY: build

# Build a beta version of stripe with the `dev` tag
build-dev:
	go generate -tags dev ./...
	go build -o stripe cmd/stripe/main.go
.PHONY: build-dev

# Build a beta version of stripe for all supported platforms
build-all-platforms:
	go generate ./...
	env GOOS=darwin go build -o stripe-darwin cmd/stripe/main.go
	env GOOS=linux go build -o stripe-linux cmd/stripe/main.go
	env GOOS=windows go build -o stripe-windows.exe cmd/stripe/main.go
.PHONY: build-all-platforms

# Build a version of stripe with the `localdev` tag
build-localdev:
	go generate ./...
	go build -o stripe -tags localdev cmd/stripe/main.go
.PHONY: build-dev

# Show to-do items per file
todo:
	@grep \
		--exclude-dir=vendor \
		--exclude-dir=node_modules \
		--exclude=Makefile \
		--text \
		--color \
		-nRo -E ' TODO:.*|SkipNow' .
.PHONY: todo

# Releases a new version
release:
# This does not release anything from your local machine but creates a tag
# for our CI to handle it

	git pull origin master

# Makefile's execute each line in its own subshell so variables don't
# persist. Instead, grab the version and run the `tag` command in the same
# subprocess by escaping the newline
	@read -p "Enter new version (of the format vN.N.N): " version; \
	git tag $$version
	git push --tags
.PHONY: release

clean:
	go clean ./...
	rm -f stripe stripe-darwin stripe-linux stripe-windows.exe
	rm -f coverage.txt
	rm -rf dist/
.PHONY: clean

# Update Node.js checksums for a specific version
# Usage: make update-node-checksums VERSION=20.18.1
update-node-checksums:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required"; \
		echo "Usage: make update-node-checksums VERSION=20.18.1"; \
		exit 1; \
	fi
	@./scripts/update-node-checksums.sh $(VERSION)
.PHONY: update-node-checksums

# Handle all protobuf generation.
protoc:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1
	@go mod tidy
	make protoc-gen-all
.PHONY: protoc

# Generate all files from protobuf definitions and compare against head.
# Because protoc comments its version in the files, ignore comments when
# diffing to avoid false positives.
protoc-ci: protoc-gen-all
	@git diff HEAD
	@git diff-index -G"^[^\/]|^[\/][^\/]" --quiet HEAD
.PHONY: proto-ci

protoc-gen-all: protoc-gen-code protoc-gen-plugin
.PHONY: protoc-gen-all

# Generate protobuf go code
protoc-gen-code:
	@protoc \
		--go_out=./rpc \
		--go_opt=module=github.com/stripe/stripe-cli/rpc \
		--go-grpc_out=require_unimplemented_servers=false:./rpc \
		--go-grpc_opt=module=github.com/stripe/stripe-cli/rpc \
		--proto_path ./rpc \
		./rpc/*.proto \
	|| (printf ${PROTOC_FAILURE_MESSAGE}; exit 1)
	@echo "Successfully compiled proto files"
.PHONY: protoc-compile

protoc-gen-plugin:
	@protoc \
		--go_out=pkg/plugins \
		--go_opt=module=github.com/stripe/stripe-cli/plugins \
		--go-grpc_out=pkg/plugins \
		--go-grpc_opt=module=github.com/stripe/stripe-cli/plugins \
	pkg/plugins/proto/main.proto \
	|| (printf ${PROTOC_FAILURE_MESSAGE}; exit 1)
	@echo "Successfully compiled proto files for plugins"
.PHONY: protoc-plugin

.DEFAULT_GOAL := build
