# StateLedger — Foundation Roadmap

> This roadmap is derived from the project README. It expands the task list into implementable workstreams. It also separates **verified facts** (directly stated in the README) from **assumptions** and **open questions** that need confirmation.

---

## 1) Verified Facts (from README)

The following statements are explicitly asserted in the README and are treated as verified requirements:

- StateLedger is an **append-only, immutable** ledger of system state needed for deterministic reconstruction.
- State at time $T$ is defined as: $S(T)=\{\text{Code, Configuration, Environment, Data Mutations}\}$.
- StateLedger is **orthogonal** to databases, message brokers, and observability tools (not a replacement).
- Collectors are **pluggable**, **non-invasive**, and **language-agnostic**.
- Four collector categories are defined: **Code**, **Configuration**, **Environment**, **Data Mutation**.
- Exact reconstruction is only possible with a **Determinism Contract**.
- If determinism cannot be guaranteed, **StateLedger fails loudly** with a precise explanation.
- Reconstruction pipeline: resolve snapshot, fetch code artifacts, load config, restore environment, replay mutations, verify checksums/invariants.
- Adoption model phases: **Passive** → **Advisory** → **Enforced**.
- Project is in **early design & MVP** stage.
- Non-goals include: replacing DBs/brokers or reconstructing arbitrary app state automatically.

---

## 2) Assumptions & Open Questions (need confirmation)

These are not defined in the README and must be decided:

- **Primary language/runtime** for the MVP service and tooling (Go, Rust, Python, Node?).
- **Storage backend** for the ledger (SQLite, Postgres, S3/object store, or append-only log like Kafka?).
- **Event ordering source** for mutations (Kafka offsets, DB tx IDs, custom monotonic counter?).
- **Cryptographic model** (hash chain vs. Merkle tree; signing model; key management).
- **Collector delivery model** (agent daemon, sidecar, library SDK, or CLI batch?).
- **Environment restoration scope** (validate only vs. full reprovisioning).
- **Target integrations** for MVP (Kafka/Postgres/Git/Docker?).
- **Schema versioning** and compatibility guarantees.
- **Multi-tenant or single-tenant** ledger assumptions.
- **Deployment model** (self-hosted only vs. cloud service later).

---

## 3) Expanded Task Breakdown (Foundation)

### Phase 0 — Product & Spec (1–2 weeks)
1. **Define MVP scope** for Passive Mode capture.
2. **Write Determinism Contract v0** (mandatory vs. optional signals).
3. **Define “state snapshot”** structure and lifecycle.
4. **Threat model**: integrity, tampering, auditability.
5. **Legal/compliance notes**: data retention, privacy, PII handling.
6. **Glossary** of terms (snapshot, mutation, collector, determinism).
7. **Out-of-scope list** (explicitly reaffirm non-goals).

### Phase 1 — Core Data Model (1–2 weeks)
6. **Ledger schema**: append-only record model.
7. **Snapshot identity**: deterministic ID + time index.
8. **Hashing strategy**: record-level + snapshot-level.
9. **Schema versioning**: version headers and compatibility rules.
10. **Error taxonomy**: missing inputs, nondeterminism, checksum mismatch, replay failure.
11. **Canonical serialization** rules (stable JSON or CBOR).
12. **Clock model** (event time vs. processing time).
13. **Ordering guarantees** definition for mutations across sources.

### Phase 2 — Storage & Integrity (2–4 weeks)
11. **Storage backend v0** (local file or SQLite for MVP).
12. **Append-only enforcement** (no mutation of prior records).
13. **Immutable sealing** (hash chain or Merkle root per snapshot).
14. **Signing model** (optional in MVP; design now).
15. **Retention policy** (configurable TTL vs. immutable archival).
16. **Backfill strategy** (ingest historical snapshots without breaking integrity).
17. **Compaction policy** (if ever allowed, define safe boundaries).
18. **Disaster recovery** plan for ledger store.

### Phase 3 — Collector Framework (2–4 weeks)
16. **Collector interface** (schema + metadata + signature).
17. **Collector SDK or CLI** (language-agnostic format + validation).
18. **Collector registry** (enable/disable, versioned).
19. **Validation rules** per collector type.
20. **Collector test harness** (fixtures + replay tests).
21. **Collector reliability** (buffering, retries, offline queue).
22. **Collector security** (identity, signing keys, rotation).
23. **Collector performance bounds** (latency/overhead budgets).

### Phase 4 — MVP Collectors (4–6 weeks)
21. **Code Collector v0**
    - Git repo ID, commit hash
    - Build artifact checksums
    - Dependency lockfiles
22. **Config Collector v0**
    - Full snapshot
    - Source & version
    - Hash and immutability enforcement
23. **Environment Collector v0**
    - OS/kernel
    - Container image hash
    - Runtime version
    - CPU arch
    - Feature flags
    - Time source
24. **Mutation Collector v0**
    - Mutation ID, order
    - Origin
    - Optional payload checksum
    - External reference (offset/tx id)
25. **Collector coverage matrix** (what each collector guarantees).
26. **Change detection** (avoid duplicates with stable hashing).

### Phase 5 — Ledger Service (4–6 weeks)
25. **Write API**: append records, validate schema.
26. **Read API**: query by time, snapshot ID.
27. **Snapshot sealing**: compute and store roots.
28. **Auth model**: service ID + signed writes (MVP may be simple tokens).
29. **Rate limiting + quotas** (optional for MVP).
30. **Idempotency model** for writes.
31. **Audit log** for write attempts and validation failures.

### Phase 6 — Reconstruction Engine (4–6 weeks)
30. **Snapshot resolution** at time $T$.
31. **Artifact fetcher** for code/config/env.
32. **Mutation ordering & replay driver**.
33. **Verification pipeline** (checksums + invariants).
34. **Forensic report** on failure.
35. **Determinism “explain why not” logic**.
36. **Partial reconstruction** mode (clearly labeled, not silent success).
37. **Reconstruction provenance** report (inputs + hashes).

### Phase 7 — Determinism Tools (3–5 weeks)
36. **Randomness capture hooks** (seed logging).
37. **Time capture** (explicit time source tracking).
38. **External dependency recording** (service versions, API responses, feature flags).
39. **Determinism confidence scoring** (Advisory Mode).
40. **Nondeterminism detectors** (e.g., time reads, RNG calls).
41. **Determinism policy config** (strict vs. permissive).

### Phase 8 — CLI & DX (2–4 weeks)
40. **CLI**: collect → push → query → reconstruct.
41. **Human-readable report format** (Markdown/JSON).
42. **Local dev mode** for rapid testing.
43. **Examples + templates** for collectors.
44. **SDK docs** with minimal integration steps.
45. **Quickstart tutorial** (end-to-end in 15 minutes).

### Phase 9 — Testing & Validation (ongoing)
44. **Unit tests** for schema + collectors.
45. **Integration tests** for end-to-end capture & reconstruct.
46. **Tamper tests** (ensure detection).
47. **Missing-state tests** (ensure “fail loudly”).
48. **Determinism violation tests**.
49. **Performance benchmarks** (write throughput, read latency).
50. **Load tests** (collector floods, burst writes).

### Phase 10 — Documentation (ongoing)
49. **Collector authoring guide**.
50. **Determinism Contract v0 spec**.
51. **Ledger schema reference**.
52. **Reconstruction guide**.
53. **Adoption guide** (Passive → Advisory → Enforced).
54. **FAQ** (what StateLedger is/is not, common misconceptions).
55. **Security model** overview.

---

## 6) Robustness Workstreams (Deep Brainstorm)

These are additional areas that strengthen the foundation beyond the MVP path.

### A) Security & Integrity
- End-to-end signing of collector payloads.
- Key rotation and revocation strategy.
- Tamper-evident ledger proofs (hash chain or Merkle audit paths).
- Secure time source validation (NTP source pinning or trusted time attestations).

### B) Privacy & Compliance
- PII tagging in snapshots.
- Redaction policy for sensitive fields.
- Data retention schedules per collector type.
- Legal export format for audits.

### C) Interoperability
- Format adapters for Kafka offsets, DB tx IDs, and message brokers.
- Pluggable artifact storage (local, S3, OCI registry).
- Config source adapters (files, env, remote config services).

### D) Reliability & Operations
- High-availability mode for the ledger service.
- Backpressure behavior for collectors under load.
- Observability for the ledger itself (metrics + tracing).
- Disaster recovery runbook.

### E) Governance & Ecosystem
- Collector certification rules (quality + compatibility).
- Versioning policy for the Determinism Contract.
- Community contribution guidelines (collectors, storage backends).

---

## 7) Metrics to Track (Success Signals)

- Percentage of requests with fully reconstructible state.
- Mean time to reconstruct a snapshot at time $T$.
- Determinism violations detected per service per week.
- Collector coverage across services (% of services with all 4 collector types).
- Reconstruction failure reasons breakdown.

---

## 8) Additional Open Questions (Operational)

- How are multiple services’ snapshots correlated at the same logical time $T$?
- How does the system define “time $T$” in multi-region deployments?
- Should the ledger support **logical time** (vector clocks) in addition to wall time?
- What is the minimum metadata required to make a snapshot “valid”?
- How are artifacts stored and referenced to avoid duplication?
- How are schema migrations handled across snapshots?


---

## 4) Suggested MVP Slice (Fastest Value)

If you want a minimal but complete foundation:

- Storage backend: **local file or SQLite**
- Collectors: **Code + Config + Environment**
- Mutation collector: **record-only** (no replay yet)
- Ledger service: **append + query**
- Reconstruction: **validate only** (no full reprovisioning)
- Output: **forensic report**

---

## 5) Next Step (Decision Needed)

Pick the MVP stack and storage target to unblock implementation. Once confirmed, the roadmap can be converted into a milestone plan with dates and owners.
