# build stage
FROM golang:1.17-bullseye as build

ARG BUILD_OS
ARG BUILD_ARCH
ARG BUILD_DATE
ARG GIT_COMMIT
ARG GIT_URL
ARG TAG

WORKDIR /src
COPY . .

RUN \
  GOOS=${BUILD_OS} GOARCH=${BUILD_ARCH} CGO_ENABLED=1 go build -tags netgo \
    -ldflags '-s -w -extldflags "-static"' \
    -ldflags "-X main.version=${TAG}" \
    -ldflags "-X main.commit=${GIT_COMMIT}" \
    -o /build/serve cmd/serve/*.go && \
  GOOS=${BUILD_OS} GOARCH=${BUILD_ARCH} CGO_ENABLED=1 go build -tags netgo \
    -ldflags '-s -w -extldflags "-static"' \
    -ldflags "-X main.version=${TAG}" \
    -ldflags "-X main.commit=${GIT_COMMIT}" \
    -o /build/check cmd/check/*.go

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && \
  apt-get install -y upx-ucl && \
  upx /build/serve && \
  upx /build/check && \
  rm -rf /var/lib/apt/lists/*

# dist stage
FROM scratch AS dist

ARG BUILD_OS
ARG BUILD_ARCH
ARG BUILD_DATE
ARG GIT_COMMIT
ARG GIT_URL
ARG TAG

LABEL maintainer="boris.huisgen@bhexpert.fr" \
      org.label-schema.build-date="$BUILD_DATE" \
      org.label-schema.name="neon" \
      org.label-schema.description="neon/${TAG} ($BUILD_OS/$BUILD_ARCH)" \
      org.label-schema.vcs-ref="$GIT_COMMIT" \
      org.label-schema.vcs-url="$GIT_URL" \
      org.label-schema.vendor="Boris HUISGEN" \
      org.label-schema.version="$TAG" \
      org.label-schema.schema-version="1.0"

COPY --from=build /build /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/serve"]
CMD []
