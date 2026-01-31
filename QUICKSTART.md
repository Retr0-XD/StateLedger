# StateLedger Quickstart

Get started with StateLedger in 5 minutes.

## Installation

### From Source

```bash
git clone https://github.com/Retr0-XD/StateLedger.git
cd StateLedger
go build -o stateledger ./cmd/stateledger
```

### Verify Installation

```bash
./stateledger
# Output: stateledger <command> [options]
```

## Basic Usage

### 1. Initialize Ledger

Create a new ledger and artifacts directory:

```bash
./stateledger init --db my-ledger.db --artifacts ./artifacts
```

### 2. Capture Your First State

Capture the current environment:

```bash
./stateledger capture -kind environment -path /tmp > env.json
cat env.json
```

The output shows your environment snapshot (OS, runtime, arch).

### 3. Store in Ledger

```bash
./stateledger collect --db my-ledger.db --kind environment \
  --payload-json "$(jq -c '.payload' env.json)"
```

### 4. Capture Configuration

```bash
echo "port: 8080
timeout: 30s" > config.yaml

./stateledger capture -kind config -path config.yaml > config.json
./stateledger collect --db my-ledger.db --kind config \
  --source config.yaml \
  --payload-json "$(jq -c '.payload' config.json)"
```

### 5. Query Records

List all records in the ledger:

```bash
./stateledger query --db my-ledger.db
```

### 6. Verify Integrity

Check hash chain integrity:

```bash
./stateledger verify --db my-ledger.db
# Output: {"ok":true,"checked":2,"timestamp":...}
```

### 7. Reconstruct State

Get a snapshot at current time:

```bash
./stateledger snapshot --db my-ledger.db
```

This shows:
- Coverage (what was captured)
- Determinism score (0-100)
- State snapshot
- Integrity proof

### 8. Determinism Advisory

Get recommendations for improving reproducibility:

```bash
./stateledger advisory --db my-ledger.db
```

### 9. Export Audit Bundle

Create an audit-ready package:

```bash
./stateledger audit --db my-ledger.db --out audit.json
cat audit.json | jq
```

## Workflow with Manifest

For batch capture, use manifests:

### Create Manifest

```bash
./stateledger manifest create --name "prod-capture" --output manifest.json
```

Edit `manifest.json` to add collectors:

```json
{
  "version": "1.0",
  "name": "prod-capture",
  "collectors": [
    {
      "kind": "code",
      "source": "."
    },
    {
      "kind": "config",
      "source": "config.yaml"
    },
    {
      "kind": "environment"
    }
  ]
}
```

### Run Manifest

```bash
./stateledger manifest run --manifest manifest.json --db my-ledger.db --source prod
```

This captures all collectors in one command and stores them in the ledger.

## Common Patterns

### CI/CD Integration

```bash
# In your CI pipeline
./stateledger init --db build-ledger.db --artifacts ./artifacts

# Capture environment
./stateledger capture -kind environment -path . > /tmp/env.json
./stateledger collect --db build-ledger.db --kind environment \
  --payload-json "$(jq -c '.payload' /tmp/env.json)"

# Capture code state
./stateledger capture -kind code -path . > /tmp/code.json
./stateledger collect --db build-ledger.db --kind code \
  --payload-json "$(jq -c '.payload' /tmp/code.json)"

# Verify and export
./stateledger verify --db build-ledger.db
./stateledger audit --db build-ledger.db --out audit-$(date +%s).json
```

### Store Build Artifacts

```bash
./stateledger artifact put --artifacts ./artifacts --file ./dist/myapp
# Output: {"path":"artifacts/abc123...","checksum":"abc123...","size":1234}
```

### Time-Travel Queries

Reconstruct state at specific time:

```bash
# Unix timestamp (e.g., 2 hours ago)
TIMESTAMP=$(date -d '2 hours ago' +%s)
./stateledger snapshot --db my-ledger.db --time $TIMESTAMP
```

## Next Steps

- Read [README.md](README.md) for full command reference
- See [CONTRIBUTING.md](CONTRIBUTING.md) for development guide
- Check [STATUS.md](STATUS.md) for feature completeness
- Review [ROADMAP.md](ROADMAP.md) for future plans

## Troubleshooting

### Git Collector Fails

Ensure you're in a git repository:
```bash
git init
git remote add origin <url>
```

### Database Locked

Only one process can write at a time. Close other connections.

### Invalid Payload

Validate JSON against schemas in `internal/collectors/`.

## Examples Repository

See [examples/](examples/) directory for:
- Kubernetes deployment manifest
- GitHub Actions workflow
- Docker build integration
- Multi-environment captures

## Support

- GitHub Issues: Bug reports and feature requests
- GitHub Discussions: Questions and community support
