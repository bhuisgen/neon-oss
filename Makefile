DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT = $(shell git describe --always)
GIT_URL = $(shell git config --get remote.origin.url)
TAG ?= dev

.PHONY: all clean build dist dist-release

all: build

clean:
	rm -f neon
	rm -f healthcheck

build:
	CGO_ENABLED=1 go build \
		-ldflags " \
			-X github.com/bhuisgen/neon/internal/app/neon.Version=${TAG} \
			-X github.com/bhuisgen/neon/internal/app/neon.Commit=${GIT_COMMIT} \
			-X github.com/bhuisgen/neon/internal/app/neon.Date=${DATE} \
		" \
		-o neon ./cmd/neon/*
	CGO_ENABLED=1 go build \
		-ldflags " \
			-X github.com/bhuisgen/neon/internal/app/neon.Version=${TAG} \
			-X github.com/bhuisgen/neon/internal/app/neon.Commit=${GIT_COMMIT} \
			-X github.com/bhuisgen/neon/internal/app/neon.Date=${DATE} \
		" \
		-o healthcheck ./cmd/healthcheck/*
