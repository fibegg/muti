.PHONY: build test lint clean docker run-test help

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY  := muti
LDFLAGS := -s -w -X main.version=$(VERSION)

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'

## build: Build the binary
build:
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/muti

## test: Run tests with race detector
test:
	go test -race -count=1 ./...

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## vet: Run go vet
vet:
	go vet ./...

## cover: Run tests with coverage report
cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## clean: Remove build artifacts
clean:
	rm -f $(BINARY) coverage.out coverage.html
	rm -rf dist/ tmp/

## docker: Build Docker image
docker:
	docker build -t $(BINARY):$(VERSION) --build-arg VERSION=$(VERSION) .

## docker-run: Run muti in Docker
docker-run:
	docker run --rm -v "$(PWD):/workspace" -w /workspace $(BINARY):$(VERSION) $(ARGS)

## smoke: Quick smoke test
smoke: build
	./$(BINARY) --help
	./$(BINARY) list-operators
	./$(BINARY) list-languages
	./$(BINARY) version

## site: Open GitHub Pages site locally
site:
	open docs/index.html

## install: Install to GOPATH/bin
install:
	CGO_ENABLED=1 go install -ldflags="$(LDFLAGS)" ./cmd/muti

## tidy: Tidy modules
tidy:
	go mod tidy

## all: lint + test + build
all: lint test build
