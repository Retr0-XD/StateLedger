# StateLedger Examples

This directory contains practical examples for integrating StateLedger into various workflows.

## Available Examples

### 1. GitHub Actions ([github-actions.yml](github-actions.yml))

Capture build state in GitHub Actions CI/CD pipeline:
- Environment capture
- Code state tracking
- Configuration snapshots
- Audit bundle generation
- Determinism checking

**Usage:**
Copy to `.github/workflows/stateledger.yml` in your repository.

### 2. Docker Build ([docker-build.sh](docker-build.sh))

Integrate StateLedger with Docker builds:
- Dockerfile tracking
- Build environment capture
- Image layer storage
- Build reproducibility verification

**Usage:**
```bash
chmod +x examples/docker-build.sh
./examples/docker-build.sh
```

### 3. Kubernetes Job ([kubernetes-job.yaml](kubernetes-job.yaml))

Run StateLedger as a Kubernetes Job:
- ConfigMap-based manifest
- Persistent storage for ledgers
- Cluster-wide state capture
- Automated audit export

**Usage:**
```bash
kubectl apply -f examples/kubernetes-job.yaml
kubectl logs job/stateledger-capture
```

## Common Patterns

### Multi-Environment Capture

```bash
# Development
stateledger manifest create --name dev --output manifests/dev.json
stateledger manifest run --manifest manifests/dev.json --db dev.db --source dev

# Staging
stateledger manifest create --name staging --output manifests/staging.json
stateledger manifest run --manifest manifests/staging.json --db staging.db --source staging

# Production
stateledger manifest create --name prod --output manifests/prod.json
stateledger manifest run --manifest manifests/prod.json --db prod.db --source prod
```

### Continuous Verification

```bash
#!/bin/bash
# verify-builds.sh - Run hourly via cron

DB_PATH="/var/lib/stateledger/ledger.db"

# Verify integrity
if ! stateledger verify --db "$DB_PATH"; then
  echo "ALERT: Ledger integrity check failed!"
  exit 1
fi

# Check determinism
SCORE=$(stateledger snapshot --db "$DB_PATH" | jq -r '.determinism_score')
if [ "$SCORE" -lt 75 ]; then
  echo "WARNING: Determinism score below threshold: $SCORE"
  stateledger advisory --db "$DB_PATH" | mail -s "Low Determinism Alert" ops@example.com
fi
```

### Artifact Archival

```bash
#!/bin/bash
# Archive build artifacts with checksums

ARTIFACTS_DIR="/var/lib/stateledger/artifacts"
ARCHIVE_DIR="/mnt/backup/stateledger"

# Store artifacts
for file in dist/*; do
  OUTPUT=$(stateledger artifact put --artifacts "$ARTIFACTS_DIR" --file "$file")
  CHECKSUM=$(echo "$OUTPUT" | jq -r '.checksum')
  echo "$file -> $CHECKSUM" >> build-manifest.txt
done

# Backup to archive
rsync -av "$ARTIFACTS_DIR/" "$ARCHIVE_DIR/$(date +%Y%m%d)/"
```

## Integration Tips

### 1. Environment Variables

```bash
export STATELEDGER_DB=/path/to/ledger.db
export STATELEDGER_ARTIFACTS=/path/to/artifacts
```

### 2. Configuration File

Create `~/.stateledger.yaml`:
```yaml
database: /home/user/.stateledger/ledger.db
artifacts: /home/user/.stateledger/artifacts
default_source: local-dev
```

### 3. Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

stateledger capture -kind code -path . > /tmp/code.json
stateledger collect --db .stateledger/ledger.db --kind code \
  --payload-json "$(jq -c '.payload' /tmp/code.json)"
```

### 4. Jenkins Pipeline

```groovy
pipeline {
    agent any
    stages {
        stage('Capture State') {
            steps {
                sh '''
                    stateledger init --db ${WORKSPACE}/ledger.db --artifacts ${WORKSPACE}/artifacts
                    stateledger capture -kind environment -path /tmp > env.json
                    stateledger collect --db ${WORKSPACE}/ledger.db --kind environment \
                        --payload-json "$(jq -c '.payload' env.json)"
                '''
            }
        }
        stage('Verify') {
            steps {
                sh 'stateledger verify --db ${WORKSPACE}/ledger.db'
            }
        }
        stage('Archive') {
            steps {
                archiveArtifacts artifacts: 'ledger.db,audit*.json', fingerprint: true
            }
        }
    }
}
```

## Need More Examples?

Open a GitHub issue describing your use case!
