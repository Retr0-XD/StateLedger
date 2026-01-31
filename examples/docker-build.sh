#!/bin/bash
set -e

# StateLedger Docker Build Integration Example
# This script captures build state during Docker image creation

LEDGER_DB="${LEDGER_DB:-/tmp/docker-build.db}"
ARTIFACTS_DIR="${ARTIFACTS_DIR:-/tmp/artifacts}"

echo "==> Initializing StateLedger"
stateledger init --db "$LEDGER_DB" --artifacts "$ARTIFACTS_DIR"

echo "==> Capturing build environment"
stateledger capture -kind environment -path /tmp > /tmp/env.json
stateledger collect --db "$LEDGER_DB" --kind environment \
  --payload-json "$(jq -c '.payload' /tmp/env.json)"

echo "==> Capturing source code state"
if [ -d .git ]; then
  stateledger capture -kind code -path . > /tmp/code.json
  stateledger collect --db "$LEDGER_DB" --kind code \
    --source "$(pwd)" \
    --payload-json "$(jq -c '.payload' /tmp/code.json)"
fi

echo "==> Capturing Dockerfile"
if [ -f Dockerfile ]; then
  stateledger capture -kind config -path Dockerfile > /tmp/dockerfile.json
  stateledger collect --db "$LEDGER_DB" --kind config \
    --source Dockerfile \
    --payload-json "$(jq -c '.payload' /tmp/dockerfile.json)"
fi

echo "==> Building Docker image"
docker build -t myapp:latest .

echo "==> Storing image layers"
docker save myapp:latest -o /tmp/myapp.tar
stateledger artifact put --artifacts "$ARTIFACTS_DIR" --file /tmp/myapp.tar

echo "==> Verifying ledger integrity"
stateledger verify --db "$LEDGER_DB"

echo "==> Generating audit bundle"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
stateledger audit --db "$LEDGER_DB" --out "audit-docker-${TIMESTAMP}.json"

echo "==> Build state captured successfully"
echo "    Ledger: $LEDGER_DB"
echo "    Artifacts: $ARTIFACTS_DIR"
echo "    Audit: audit-docker-${TIMESTAMP}.json"
