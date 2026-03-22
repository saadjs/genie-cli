VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/genie ./cmd/genie

test:
	go test -v ./...

clean:
	rm -rf bin/

release:
	goreleaser release --clean

.PHONY: build test clean release
