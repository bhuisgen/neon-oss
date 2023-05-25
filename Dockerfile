FROM scratch

COPY share/certs/ca-certificates.crt /etc/ssl/certs/
COPY neon /

ENTRYPOINT ["/neon"]
