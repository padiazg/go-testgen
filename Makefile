.PHONY: build test lint install clean

build:
	go build -o bin/testgen ./cmd/testgen

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...

install:
	go install ./cmd/testgen

clean:
	rm -rf bin/