# Build stage
FROM golang:1.25.4 AS builder

WORKDIR /src
COPY . .

RUN go build -o /out/stateledger ./cmd/stateledger

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /out/stateledger /usr/local/bin/stateledger

ENTRYPOINT ["/usr/local/bin/stateledger"]
