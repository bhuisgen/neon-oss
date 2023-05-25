DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT = $(shell git describe --always)
GIT_URL = $(shell git config --get remote.origin.url)
TAG ?= dev

all: build

.PHONY: build
build: init
	CGO_ENABLED=1 go build -tags netgo,osusergo \
		-ldflags " \
			-s -w -extldflags '-static' \
			-X github.com/bhuisgen/neon/internal/app.version=${TAG} \
			-X github.com/bhuisgen/neon/internal/app.commit=${GIT_COMMIT} \
			-X github.com/bhuisgen/neon/internal/app.Date=${DATE} \
		" \
		-o neon cmd/exec/*.go

.PHONY: clean
clean:

.PHONY: init
init:

.PHONY: dist
dist:
	docker run --rm -v devcontainer_neon:/workspace -w /workspace/neon \
		-e DOCKER_CERT_PATH=${DOCKER_CERT_PATH} -e DOCKER_HOST=${DOCKER_HOST} -e DOCKER_TLS_VERIFY=${DOCKER_TLS_VERIFY} \
		bhuisgen/goreleaser-cross:v1.19.3-amd64 --rm-dist --snapshot 

.PHONY: dist-release
dist-release:
	docker run --rm -v devcontainer_neon:/workspace -w /workspace/neon \
		-e DOCKER_CERT_PATH=${DOCKER_CERT_PATH} -e DOCKER_HOST=${DOCKER_HOST} -e DOCKER_TLS_VERIFY=${DOCKER_TLS_VERIFY} \
		-e GITHUB_TOKEN=${GITHUB_TOKEN} \
		bhuisgen/goreleaser-cross:v1.19.3-amd64 --rm-dist