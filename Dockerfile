# Build stage
FROM golang:1.25.4 AS builder

WORKDIR /src
COPY . .

# Build using the vendored modules so the build is reproducible and does not
# depend on network access to the Go module proxy.
RUN go build -mod=vendor -o /out/stateledger ./cmd/stateledger

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -r -u 10001 -g root stateledger

WORKDIR /app
COPY --from=builder /out/stateledger /usr/local/bin/stateledger

# Persist the ledger database outside the container image.
VOLUME ["/app/data"]
ENV STATELEDGER_DB=/app/data/ledger.db

# Run as a non-root user for the principle of least privilege.
USER 10001

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/usr/local/bin/stateledger", "query", "--id", "0"] || exit 1

# Default command: start the HTTP API server. Override DB path via STATELEDGER_DB.
ENTRYPOINT ["/usr/local/bin/stateledger"]
CMD ["server", "--db", "/app/data/ledger.db", "--addr", ":8080"]
