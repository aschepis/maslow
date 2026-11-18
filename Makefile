.PHONY: build test vet lint validate verify release clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
PKG     := github.com/aschepis/maslow/internal/buildinfo
LDFLAGS := -X $(PKG).version=$(VERSION) -X $(PKG).gitSHA=$(COMMIT)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/maslow ./cmd/maslow/

test:
	go test ./...

vet:
	go vet ./...

lint: vet
	cue vet schema/maslow.cue

validate: build
	bin/maslow validate maslow.yaml

verify: build
	bin/maslow verify --profile full

release:
	scripts/build-release.sh $(VERSION)

clean:
	rm -rf bin/ dist/ reports/
