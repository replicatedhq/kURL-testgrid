SHELL := /bin/bash -o pipefail
CURRENT_USER=$(shell id -u -n)


.PHONY: all
all: fmt vet test build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run --enable-all ./...

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
		-o ./bin/tgrun \
		./cmd/run

.PHONY: run
run:
	./bin/tgrun

ttl-sh:
	docker build -f Dockerfile \
		--build-arg gitsha=test \
		--build-arg buildtime="$(date --rfc-3339=seconds | sed 's/ /T/')" \
		-t ttl.sh/${CURRENT_USER}/tgrun:12h \
		..
	docker push ttl.sh/${CURRENT_USER}/tgrun:12h
