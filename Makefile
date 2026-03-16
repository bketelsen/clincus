BINARY_NAME := clincus
MODULE := github.com/bketelsen/clincus
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
	-X $(MODULE)/internal/cli.Version=$(VERSION) \
	-X $(MODULE)/internal/cli.Commit=$(COMMIT) \
	-X $(MODULE)/internal/cli.Date=$(BUILD_DATE)

.PHONY: build web test lint fmt completions manpages bump docs docs-serve install clean

build: web
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/clincus

web:
	cd web && npm install && npm run build

test:
	go test -race -v ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

completions:
	mkdir -p completions
	go run ./cmd/clincus completion bash > completions/clincus.bash
	go run ./cmd/clincus completion zsh > completions/clincus.zsh
	go run ./cmd/clincus completion fish > completions/clincus.fish

manpages:
	mkdir -p manpages
	go run ./cmd/clincus man --dir manpages
	gzip -f manpages/*.1

bump: build test fmt lint
	@test -z "$$(git status --porcelain)" || (echo "Working directory not clean" && exit 1)
	@VERSION=$$(svu next) && \
		git tag -a $$VERSION -m "$$VERSION" && \
		git push origin $$VERSION

docs:
	mkdocs build

docs-serve:
	mkdocs serve

install: build
	install -m 755 $(BINARY_NAME) ~/.local/bin/$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME)
	rm -rf webui/dist/assets webui/dist/index.html
	rm -rf completions manpages
