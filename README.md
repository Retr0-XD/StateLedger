2. Deterministic State Reconstruction (Also unsolved)
Problem

After an incident:

You can replay events

You cannot reconstruct exact system state

Config + env + timing are missing

Kafka + event sourcing ≠ enough.

Missing primitive

“Given time T, reconstruct system state exactly.”

What to build

A State Ledger:

Stores:

Config

Env

Code version

Data mutations

Guarantees replayable state

This is orthogonal to Kafka/DBs.

Why this matters

Forensics

Compliance

AI auditability

Disaster recovery beyond backups
