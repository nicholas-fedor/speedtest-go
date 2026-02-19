BINARY_NAME=speedtest-go
VERSION ?= dev
COMMIT = $(shell git rev-parse --short HEAD)
DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

.PHONY: all build test clean lint fmt vet install release docker-build examples deps generate build-all test-ci ci mocks setup-ci

all: build

build:
	go build -ldflags "-s -w -X github.com/nicholas-fedor/speedtest-go/speedtest.version=$(VERSION) -X github.com/nicholas-fedor/speedtest-go/internal/output.commit=$(COMMIT) -X github.com/nicholas-fedor/speedtest-go/internal/output.date=$(DATE)" -o bin/$(BINARY_NAME) .

test:
	go test ./...

clean:
	rm -f $(BINARY_NAME)

lint:
	golangci-lint run --fix --config build/golangci-lint/golangci-lint.yaml

fmt:
	go fmt ./...

vet:
	go vet

install:
	go install -ldflags "-s -w -X github.com/nicholas-fedor/speedtest-go/speedtest.version=$(VERSION) -X github.com/nicholas-fedor/speedtest-go/internal/output.commit=$(COMMIT) -X github.com/nicholas-fedor/speedtest-go/internal/output.date=$(DATE)" .

release:
	goreleaser release --clean --config build/goreleaser/goreleaser.yaml

docker-build:
	docker build -t $(BINARY_NAME) .

examples:
	go build ./example/...

deps:
	go mod download

generate:
	go generate ./...

build-all:
	go build ./...

test-ci:
	go test ./speedtest -v

ci: deps generate build-all test-ci lint

setup-ci: deps generate fmt vet

mocks: ## Generate mock files for testing
	mockery --config build/mockery/mockery.yaml
