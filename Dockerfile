FROM alpine:3.16 AS base

FROM scratch

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY neon /

ENTRYPOINT ["/neon"]
