DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT = $(shell git describe --always)
GIT_URL = $(shell git config --get remote.origin.url)
TAG ?= dev

.PHONY: all clean build dist dist-release

all: build

clean:
	rm -fr build/

build:
	CGO_ENABLED=1 go build -tags netgo,osusergo \
		-ldflags " \
			-s -w -extldflags '-static' \
			-X github.com/bhuisgen/neon/internal/app/neon.Version=${TAG} \
			-X github.com/bhuisgen/neon/internal/app/neon.Commit=${GIT_COMMIT} \
			-X github.com/bhuisgen/neon/internal/app/neon.Date=${DATE} \
		" \
		-o neon ./cmd/neon/*

dist:
	docker run --rm -v ${DOCKER_VOLUME}:/workspace -w /workspace/neon-oss \
		-e DOCKER_CERT_PATH=${DOCKER_CERT_PATH} -e DOCKER_HOST=${DOCKER_HOST} -e DOCKER_TLS_VERIFY=${DOCKER_TLS_VERIFY} \
		goreleaser/goreleaser-cross:v1.21 --snapshot --clean

dist-release:
	docker run --rm -v ${DOCKER_VOLUME}:/workspace -w /workspace/neon-oss \
		-e DOCKER_CERT_PATH=${DOCKER_CERT_PATH} -e DOCKER_HOST=${DOCKER_HOST} -e DOCKER_TLS_VERIFY=${DOCKER_TLS_VERIFY} \
		-e GITHUB_TOKEN=${GITHUB_TOKEN} \
		goreleaser/goreleaser-cross:v1.21 --clean
