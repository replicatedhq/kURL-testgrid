SHELL := /bin/bash

.PHONY: all
all: fmt vet test build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	golangci-lint run ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test -v ./...

.PHONY: build
build:
	mkdir -p bin
	go build \
		-o ./bin/tgapi \
		./cmd/api

.PHONY: run
run:
	./bin/tgapi run
