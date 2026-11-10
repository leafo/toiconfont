.PHONY: build install test clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILT_AT ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS = -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.builtAt=$(BUILT_AT)

build:
	go build -ldflags "$(LDFLAGS)" -o toiconfont .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

clean:
	go clean ./...
	rm -f toiconfont
