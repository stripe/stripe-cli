export GO111MODULE := on
export GOBIN := $(shell pwd)/bin
export PATH := $(GOBIN):$(PATH)

all: test

install-deps:
	go get
.PHONY: install-deps

update-deps:
	go get -u
.PHONY: update-deps

update-openapi-spec:
	rm -f ./api/openapi-spec/spec3.sdk.json
	wget https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.sdk.json -P ./api/openapi-spec
.PHONY: update-openapi-spec

lint:
# In travis, we need to install golint explicitly. Don't do this in other
# environments
ifeq ($(ENVIRONMENT), travis)
	go get golang.org/x/lint/golint
	git checkout .
endif
	golint -set_exit_status ./...
.PHONY: lint

vet:
	go vet $(shell go list ./... | grep -v /vendor/)
.PHONY: vet

test: install-deps lint vet
	go test -race -cover -v ./...
	@echo '\o/ yay, we did it!'
.PHONY: test

build:
	go mod download
	go generate ./...
	go build -o stripe -ldflags "-s -w" cmd/stripe/main.go
.PHONY: build

# This does not release anything from your local machine but creates a tag
# for our CI to handle it
release:
	git pull origin master

# Makefile's execute each line in its own subshell so variables don't
# persist. Instead, grab the version and run the `tag` command in the same
# subprocess by escaping the newline
	@read -p "Enter new version (of the format vN.N.N): " version; \
	git tag $$version
	git push --tags
.PHONY: release
