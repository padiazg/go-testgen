.PHONY: build test lint install clean

build: pkg=github.com/padiazg/hexago/pkg/version
build: ldflags = -X $(pkg).version=$(shell git describe --tags --always --dirty) 
build: ldflags += -X $(pkg).commit=$(shell git rev-parse HEAD)
build: ldflags += -X $(pkg).buildDate=$(shell date -Iseconds)

build:
	@echo "Building go-testgen..."
	@go build -o go-testgen -ldflags "$(ldflags)"

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/