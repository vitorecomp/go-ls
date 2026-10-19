PREFIX ?=
BINDIR ?=
DATADIR ?=

.PHONY: all build build-web build-go install install-system install-go uninstall test vet fmt clean

all: build

build-web:
	cd web && npm install --no-audit --no-fund && npm run build

build-go:
	go build -ldflags="-s -w" -o go-ls .
	go build -ldflags="-s -w" -o go-ls-server ./cmd/server
	go build -ldflags="-s -w" -o go-ls-client ./cmd/client

build: build-web build-go

install:
	PREFIX="$(PREFIX)" BINDIR="$(BINDIR)" DATADIR="$(DATADIR)" ./scripts/install.sh

install-system:
	PREFIX="$(PREFIX)" BINDIR="$(BINDIR)" DATADIR="$(DATADIR)" ./scripts/install.sh --system

install-go:
	go install . ./cmd/server ./cmd/client

uninstall:
	PREFIX="$(PREFIX)" BINDIR="$(BINDIR)" DATADIR="$(DATADIR)" ./scripts/install.sh --uninstall

fmt:
	gofmt -s -w .
	go mod tidy

vet:
	go vet ./...

test: vet
	go test -v -race ./...

clean:
	rm -f go-ls go-ls-server go-ls-client
	rm -rf web/out web/.next
