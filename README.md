# StateLedger

Deterministic system state reconstruction with cryptographic integrity proofs. Capture code, configuration, environment, and mutations into an append-only ledger, then reconstruct and audit state at any point in time.

---

## What it does

- **Capture** system state (code/config/environment/mutations)
- **Store** in an immutable, append-only ledger
- **Verify** integrity via SHA-256 hash chains
- **Reconstruct** state at time T
- **Export** audit bundles for compliance

---

## Quick start (local)

Build:

```bash
go build -o stateledger ./cmd/stateledger
```

Initialize:

```bash
./stateledger init --db data/ledger.db --artifacts artifacts
```

Capture + collect environment:

```bash
./stateledger capture -kind environment -path /tmp | jq -c '.payload' | \
  xargs -I {} ./stateledger collect --db data/ledger.db --kind environment --payload-json '{}'
```

Verify + snapshot:

```bash
./stateledger verify --db data/ledger.db
./stateledger snapshot --db data/ledger.db
```

Export audit bundle:

```bash
./stateledger audit --db data/ledger.db --out audit.json
```

---

## CLI commands

| Command | Purpose |
| --- | --- |
| `init` | Initialize ledger DB and artifacts store |
| `capture` | Run collectors (code/config/environment) |
| `collect` | Validate and append payloads |
| `manifest` | Batch capture (create/run/show) |
| `append` | Append record directly |
| `query` | Query ledger records |
| `verify` | Verify hash chain integrity |
| `snapshot` | Reconstruct state at time T |
| `advisory` | Determinism analysis | 
| `audit` | Export audit bundle |
| `artifact put` | Store artifact by checksum |

---

## Docker & Kubernetes usage

Yes — this project works well as a **batch job** in Kubernetes. It is designed to run as a **CLI or Job/CronJob**, not as a long-running server. Typical integration:

1. Use a **PVC** for the ledger DB and artifacts directory.
2. Run a **Job/CronJob** to capture state and export audits.
3. Store `ledger.db` and audit bundles in durable storage.

Example Kubernetes job: [examples/kubernetes-job.yaml](examples/kubernetes-job.yaml)

---

## Docker image

This repo supports building and pushing a Docker image to Docker Hub using GitHub Actions.

### Required GitHub Secrets

Set these in your repo settings:

- `DOCKERHUB_USERNAME`
- `DOCKERHUB_TOKEN`
- `DOCKERHUB_REPO` (e.g. `yourname/stateledger`)

### Build and Push (CI)

The workflow builds and pushes images on every push to `main`:

- `latest`
- `sha-<commit>`

---

## Integration patterns

### CI/CD

- Capture environment + code during builds
- Export audit bundles
- Store artifacts with checksums

See: [examples/github-actions.yml](examples/github-actions.yml)

### Docker builds

Capture build state during image creation:

See: [examples/docker-build.sh](examples/docker-build.sh)

---

## Determinism and reproducibility

StateLedger computes a **determinism score (0–100)** based on how complete the captured state is. Missing code/config/environment lowers reproducibility.

Use:

```bash
./stateledger advisory --db data/ledger.db
```

---

## Documentation

- Quickstart: [QUICKSTART.md](QUICKSTART.md)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
- Status: [STATUS.md](STATUS.md)
- Examples: [examples/README.md](examples/README.md)

---

## License

Apache 2.0 — see [LICENSE](LICENSE).

# StateLedger

**Deterministic System State Reconstruction for Distributed Systems**

---

## Overview

Modern systems can replay events, restore databases, and roll back code — yet **cannot reconstruct exact system state** at a given point in time.

After incidents, audits, or failures, teams still ask:

> *“What exactly was the system state at time T — and can we prove it?”*

**StateLedger** is an open-source infrastructure primitive that solves this problem.

It provides a **time-addressable, deterministic record of system state** that allows exact reconstruction of declared system state at any point in time, across heterogeneous systems.

StateLedger is **orthogonal** to databases, message brokers, and observability tools.
It does not replace Kafka, Postgres, or Git — it **binds them together into a single source of state truth**.

---

## The Problem

Today’s systems suffer from a fundamental gap:

### What we can do

* Replay events (Kafka, event sourcing)
* Restore data (backups, snapshots)
* Roll back code (Git)
* Inspect logs and traces

### What we cannot do

* Reconstruct *exact* system state at time T
* Prove why a system behaved a certain way
* Reproduce AI or business decisions deterministically
* Perform complete forensic or compliance audits

This gap exists because **system state is fragmented** across:

* Code
* Configuration
* Runtime environment
* Data mutations
* Time and nondeterminism

No existing system captures all of these **together**, in a verifiable and replayable way.

---

## Core Idea

StateLedger introduces a new infrastructure abstraction:

> **A State Ledger** — an append-only, immutable record of everything required to deterministically reconstruct declared system state.

At time `T`, system state is defined as:

```
S(T) = {
  Code,
  Configuration,
  Environment,
  Data Mutations
}
```

If all four dimensions are available and deterministic, **exact reconstruction is guaranteed**.

If any dimension is missing or nondeterministic, StateLedger explicitly reports **why reconstruction is impossible** — never silently wrong.

---

## What StateLedger Is (and Is Not)

### StateLedger **IS**

* A system-level **state truth ledger**
* A deterministic **replay authority**
* A forensic and compliance foundation
* A unifying layer across existing infrastructure

### StateLedger **IS NOT**

* A database
* A message broker
* A workflow engine
* An observability platform
* A replacement for Kafka, Redis, Git, or Kubernetes

---

## Architecture

```
┌────────────────────────────────────┐
│            State Ledger             │
│   (append-only, immutable store)    │
└───────────────┬────────────────────┘
                │
┌───────────────┴────────────────────┐
│          State Collectors           │
│  (pluggable, non-intrusive agents)  │
└───────────────┬────────────────────┘
                │
┌───────────────┴────────────────────┐
│        State Reconstructor          │
│ (deterministic replay & verifier)  │
└────────────────────────────────────┘
```

---

## State Collectors

StateLedger achieves universality through **pluggable, declarative collectors**.
Collectors are **non-invasive** and **language-agnostic**.

### 1. Code Collector

Records:

* Repository identifier
* Commit hash
* Build artifact checksums
* Dependency lockfiles

Example:

```json
{
  "code": {
    "repo": "orders-service",
    "commit": "a7c91e",
    "artifacts": ["sha256:3f2a..."]
  }
}
```

---

### 2. Configuration Collector

Records:

* Complete config snapshot
* Source (file, env, remote)
* Version
* Cryptographic hash

Rules:

* Config snapshots are immutable
* Partial config is explicitly rejected

---

### 3. Environment Collector

Records runtime context:

* OS and kernel
* Container image hash
* Runtime (JVM, Node, Python, etc.)
* CPU architecture
* Feature flags
* Time source

Example:

```json
{
  "environment": {
    "os": "linux",
    "container": "sha256:ab91...",
    "runtime": "jvm-21",
    "flags": ["FAST_PATH=true"]
  }
}
```

---

### 4. Data Mutation Collector

Does **not** replace Kafka or databases.

Records:

* Mutation identity
* Order
* Origin
* Payload checksum (optional)
* External reference (Kafka offset, DB tx id)

Example:

```json
{
  "mutation": {
    "type": "order_created",
    "id": "evt-91823",
    "source": "orders-service",
    "hash": "sha256:f19c..."
  }
}
```

Kafka stores the data.
StateLedger stores the **truth of mutation occurrence**.

---

## Determinism Contract

Exact reconstruction is only possible if systems obey a **Determinism Contract**.

### Required guarantees

* No hidden randomness
* No hidden time dependencies
* External calls are declared
* Inputs are versioned

### How StateLedger handles violations

* Random seeds are captured
* Time is virtualized where possible
* External dependencies are recorded
* Violations are surfaced explicitly

If determinism cannot be guaranteed:

> StateLedger fails loudly with a precise explanation.

---

## Reconstruction

Reconstructing state at time `T`:

```
1. Resolve ledger snapshot at T
2. Fetch code version and artifacts
3. Load configuration snapshot
4. Restore environment
5. Replay mutations in order
6. Verify checksums and invariants
```

If verification fails, reconstruction halts with a **forensic report**.

---

## Why Kafka + Event Sourcing Is Not Enough

| Capability             | Kafka | DB | StateLedger |
| ---------------------- | ----- | -- | ----------- |
| Event replay           | ✓     | ✗  | ✓           |
| Config replay          | ✗     | ✗  | ✓           |
| Environment replay     | ✗     | ✗  | ✓           |
| Determinism guarantees | ✗     | ✗  | ✓           |
| State proof            | ✗     | ✗  | ✓           |

Kafka handles **data flow**.
StateLedger handles **state truth**.

---

## Adoption Model

StateLedger is designed for **incremental adoption**.

### Phase 1: Passive Mode (MVP)

* Collect state snapshots
* No enforcement
* Forensics and audits only

### Phase 2: Advisory Mode

* Detect nondeterminism
* Warn about missing state
* Provide reconstruction confidence scores

### Phase 3: Enforced Mode

* CI/CD gating
* Compliance guarantees
* Regulated environments

---

## Use Cases

### Incident Forensics

Reconstruct exact system state during an outage or data corruption event.

### Compliance & Audit

Prove how a system behaved at a given time — with evidence.

### AI Auditability

Reproduce model outputs with the same:

* Prompt
* Model version
* Environment
* Data

### Disaster Recovery

Restore **correctness**, not just data.

---

## Design Principles

* Append-only, immutable core
* Explicit over implicit
* Determinism over convenience
* Failure transparency
* Orthogonal to existing systems

---

## Non-Goals

* Automatic capture of arbitrary application state
* Business logic interpretation
* Replacing databases or brokers
* Magic reconstruction without determinism

---

## Why This Project Exists

This problem cannot be solved by:

* Configuration
* Plugins
* Observability
* Better logging

It requires a **new infrastructure primitive**.

StateLedger exists to fill that gap.

---

## Status

🚧 Early design & MVP phase
Contributions welcome — especially:

* Collector implementations
* Determinism enforcement strategies
* Storage backends
* Formal specifications

---

## Quickstart (Local, Self-Contained)

StateLedger is intentionally built to run on any system with Go installed.
It uses a **pure-Go SQLite driver** that is vendored in this repository to avoid external system dependencies.

### Build

```
go build -o stateledger ./cmd/stateledger
```

### Initialize

```
./stateledger init --db data/ledger.db --artifacts artifacts
```

### Append a record

```
./stateledger append --type code --source orders-service --payload-json '{"commit":"a7c91e"}'
```

### Collect (validated payloads)

```
./stateledger collect --kind code --payload-json '{"repo":"orders","commit":"a7c91e"}'
./stateledger collect --kind config --payload-json '{"source":"env","version":"1","hash":"sha256:...","snapshot":"KEY=VALUE"}'
./stateledger collect --kind environment --payload-json '{"os":"linux","runtime":"go1.25","arch":"amd64","time_source":"system"}'
```

### Capture (real collectors — Git, Env, Config)

```
./stateledger capture --kind code --path .
./stateledger capture --kind environment --path ""
./stateledger capture --kind config --path config.json
```

### Manifest (batch capture)

Create a manifest:
```
./stateledger manifest create --name "my-app" --output manifest.json
```

Edit `manifest.json` as needed, then run:
```
./stateledger manifest run --file manifest.json --db data/ledger.db
```

Show manifest:
```
./stateledger manifest show --file manifest.json
```

### Snapshot (reconstruction summary)

```
./stateledger snapshot --db data/ledger.db --time 0
```

### Advisory (determinism analysis)

```
./stateledger advisory --db data/ledger.db --time 0
```

### Audit Bundle (export)

```
./stateledger audit --db data/ledger.db --time 0 --out audit.json
```

### Query records

```
./stateledger query --since 0 --limit 100
```

### Verify ledger integrity

```
./stateledger verify
```

### Store an artifact

```
./stateledger artifact put --file ./path/to/artifact.bin
```

---

## CLI Commands

| Command | Purpose |
| --- | --- |
| `init` | Initialize the SQLite ledger and artifact store. |
| `collect` | Validate collector payloads and append to the ledger. |
| `capture` | Execute real collectors (Git, Env, Config) and output payload. |
| `manifest` | Create, show, and run batch capture manifests. |
| `append` | Append an immutable record to the ledger. |
| `query` | Fetch records by ID or time range. |
| `verify` | Verify the hash chain integrity. |
| `snapshot` | Resolve and summarize state at a given time. |
| `advisory` | Run determinism advisory analysis for a time range. |
| `audit` | Export a snapshot + proof bundle for auditing. |
| `artifact put` | Store an artifact by checksum. |

---

## Architecture (MVP Implementation)

**StateLedger Core:**
- Append-only SQLite ledger with hash chain integrity
- Immutable record storage (code, config, environment, mutations)
- Verification via VerifyChain (detects tampering)

**Collectors (Real Implementations):**
- Code Collector: Extracts Git repo name, commit hash via `git` CLI
- Environment Collector: Captures OS, runtime, arch via `runtime` package
- Config Collector: Reads file snapshots and computes SHA-256 hash
- Mutation Collector: Records external references (Kafka offsets, DB tx IDs)

**Manifest Format:**
- JSON-based declarative batch capture specification
- Versioning for schema compatibility
- Pluggable collector parameters

**CLI:**
- `collect` — validate and ingest structured payloads
- `capture` — invoke real collectors automatically
- `manifest` — batch capture workflows
- `query/verify` — ledger inspection and integrity checks

---

## Testing

### Unit Tests

Run all unit tests across packages:

```bash
go test ./...
```

Test coverage by package:
- **collectors** - Payload validation (code/config/environment/mutation schemas)
- **manifest** - Manifest format parsing and validation
- **sources** - Real collectors (Git/Environment/Config capture)
- **artifacts** - Content-addressable artifact storage
- **ledger** - Append-only ledger, hash chain verification, reconstruction engine
- **cmd/stateledger** - CLI integration tests (full workflow)

### Integration Tests

The CLI package includes comprehensive integration tests that verify:
- Database initialization
- Manifest creation and execution
- Collector capture and data ingestion
- Hash chain verification
- Snapshot reconstruction at time T
- Determinism advisory analysis
- Audit bundle export
- Artifact storage and retrieval

Run CLI integration tests specifically:

```bash
go test ./cmd/stateledger -v
```

### Smoke Test (Manual)

Quick end-to-end verification:

```bash
go build -o /tmp/stateledger ./cmd/stateledger
/tmp/stateledger init --db /tmp/sl/ledger.db --artifacts /tmp/sl/artifacts
/tmp/stateledger manifest create --name "smoke" --output /tmp/sl/manifest.json
/tmp/stateledger manifest run --file /tmp/sl/manifest.json --db /tmp/sl/ledger.db --source smoke
/tmp/stateledger verify --db /tmp/sl/ledger.db
/tmp/stateledger snapshot --db /tmp/sl/ledger.db --time 0
/tmp/stateledger audit --db /tmp/sl/ledger.db --time 0 --out /tmp/sl/audit.json
```

---

## License

Apache 2.0

---
