# build stage
FROM golang:1.18-bullseye as build

ARG BUILD_OS
ARG BUILD_ARCH
ARG BUILD_DATE
ARG GIT_COMMIT
ARG GIT_URL
ARG TAG

WORKDIR /src
COPY . .
RUN GOOS=${BUILD_OS} GOARCH=${BUILD_ARCH} CGO_ENABLED=1 go build -tags netgo \
      -ldflags "-s -w -extldflags '-static' -X main.version=${TAG} -X main.commit=${GIT_COMMIT}" \
      -o /build/neon cmd/exec/*.go

# compress stage
FROM bhuisgen/alpine-go:prod AS compress

RUN apk add --no-cache upx

COPY --from=build /build/ /build/

RUN upx /build/neon

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
      org.label-schema.description="neon/$TAG ($BUILD_OS/$BUILD_ARCH)" \
      org.label-schema.vcs-ref="$GIT_COMMIT" \
      org.label-schema.vcs-url="$GIT_URL" \
      org.label-schema.vendor="Boris HUISGEN" \
      org.label-schema.version="$TAG" \
      org.label-schema.schema-version="1.0"

COPY --from=compress /build /
COPY --from=compress /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/neon"]
CMD []
