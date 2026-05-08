BINARY_NAME=speedtest-go
VERSION ?= dev
COMMIT = $(shell git rev-parse --short HEAD)
DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
NFPM_CONFIG ?= build/nfpm/nfpm.yaml
NFPM_TARGET ?= dist
NFPM_GOARCH ?= amd64

.PHONY: all build test clean lint fmt vet install release docker-build examples deps generate build-all test-ci ci mocks setup-ci nfpm-deb nfpm-rpm nfpm-apk nfpm-arch nfpm-all

all: build

build:
	go build -ldflags "-s -w -X github.com/nicholas-fedor/speedtest-go/speedtest.version=$(VERSION) -X github.com/nicholas-fedor/speedtest-go/internal/output.commit=$(COMMIT) -X github.com/nicholas-fedor/speedtest-go/internal/output.date=$(DATE)" -o bin/$(BINARY_NAME) .

test:
	go test -v -covermode atomic ./...

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
	goreleaser release --clean --config build/goreleaser/stable.yaml

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

nfpm-deb: build ## Build a .deb package locally
	@mkdir -p $(NFPM_TARGET)
	GOARCH=$(NFPM_GOARCH) VERSION=$(VERSION) nfpm pkg --config $(NFPM_CONFIG) --packager deb --target $(NFPM_TARGET)/

nfpm-rpm: build ## Build an .rpm package locally
	@mkdir -p $(NFPM_TARGET)
	GOARCH=$(NFPM_GOARCH) VERSION=$(VERSION) nfpm pkg --config $(NFPM_CONFIG) --packager rpm --target $(NFPM_TARGET)/

nfpm-apk: build ## Build an .apk package locally
	@mkdir -p $(NFPM_TARGET)
	GOARCH=$(NFPM_GOARCH) VERSION=$(VERSION) nfpm pkg --config $(NFPM_CONFIG) --packager apk --target $(NFPM_TARGET)/

nfpm-arch: build ## Build an Archlinux package locally
	@mkdir -p $(NFPM_TARGET)
	GOARCH=$(NFPM_GOARCH) VERSION=$(VERSION) nfpm pkg --config $(NFPM_CONFIG) --packager archlinux --target $(NFPM_TARGET)/

nfpm-all: nfpm-deb nfpm-rpm nfpm-apk nfpm-arch ## Build all Linux packages locally
