DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT = $(shell git describe --always)
GIT_URL = $(shell git config --get remote.origin.url)
TAG ?= dev

.PHONY: all build clean init dev

all: build

build: clean init
	docker build --build-arg BUILD_OS=linux --build-arg BUILD_ARCH=amd64 --build-arg TAG=$(TAG) \
		--build-arg BUILD_DATE=$(DATE) --build-arg GIT_COMMIT=$(GIT_COMMIT) --build-arg GIT_URL=$(GIT_URL) \
		-t neon:$(TAG) .

clean:

init:

dev:
	docker build --build-arg BUILD_OS=linux --build-arg BUILD_ARCH=amd64 -t neon:dev -f Dockerfile.dev .
	docker run -it --rm  --name debug $(OPTIONS) neon:dev dap --listen=:12345
