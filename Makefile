SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=

export GO111MODULE := on
export GOBIN := $(shell pwd)/bin
export PATH := $(GOBIN):$(PATH)
export GOPROXY := https://gocenter.io

# Install all the build and lint dependencies
setup:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh
	curl -L https://git.io/misspell | sh
	go mod download
.PHONY: setup

# Run all the tests
test:
	go test $(TEST_OPTIONS) -failfast -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=2m
.PHONY: test

# Run all the tests and opens the coverage report
cover: test
	go tool cover -html=coverage.txt
.PHONY: cover

# gofmt and goimports all go files
fmt:
	find . -name '*.go' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done
.PHONY: fmt

# Run all the linters
lint:
	# TODO: fix disabled linter issues
	./bin/golangci-lint run ./...
	./bin/misspell -error **/*.go
.PHONY: lint

# Clean go.mod
go-mod-tidy:
	@go mod tidy -v
	@git diff HEAD
	@git diff-index --quiet HEAD
.PHONY: go-mod-tidy

# Run all the tests and code checks
ci: build test lint go-mod-tidy
.PHONY: ci

# Build a beta version of stripe
build:
	go generate ./...
	go build -o stripe cmd/stripe/main.go
.PHONY: build

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

# Updates the OpenAPI spec
update-openapi-spec:
	rm -f ./api/openapi-spec/spec3.sdk.json
	wget https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.sdk.json -P ./api/openapi-spec
.PHONY: update-openapi-spec

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

.DEFAULT_GOAL := build
