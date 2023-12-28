FROM scratch

COPY neon /
COPY healthcheck /
COPY share/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/neon"]
