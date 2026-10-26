VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/genie ./cmd/genie

install:
	go install $(LDFLAGS) ./cmd/genie

test:
	go test -race ./...

check:
	@test -z "$$(gofmt -l .)" || (gofmt -l .; echo "run: gofmt -w ." && exit 1)
	go vet ./...
	go test -race ./...

clean:
	rm -rf bin/

release:
	goreleaser release --clean

.PHONY: build install test check clean release
