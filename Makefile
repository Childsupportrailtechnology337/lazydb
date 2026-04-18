BINARY_NAME=lazydb
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")

.PHONY: build run clean test fmt lint deps install release

build:
	go build -ldflags "-s -w -X github.com/aymenworks/lazydb/cmd.version=$(VERSION)" -o $(BINARY_NAME) .

run:
	go run . $(ARGS)

clean:
	rm -f $(BINARY_NAME)

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

deps:
	go mod tidy

install:
	go install -ldflags "-s -w -X github.com/aymenworks/lazydb/cmd.version=$(VERSION)" .

release:
	goreleaser release --clean

release-snapshot:
	goreleaser release --snapshot --clean
