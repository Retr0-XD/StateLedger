# StateLedger Performance Benchmarks

Performance characteristics of StateLedger core operations.

## Benchmark Results

Results from AMD EPYC 7763 64-Core Processor (2 cores allocated):

### Ledger Operations

| Operation | Ops/sec | ns/op | Memory/op | Allocs/op |
|-----------|---------|-------|-----------|-----------|
| **Append** | ~11,830 | 84,537 | 1,520 B | 41 |
| **AppendParallel** | ~11,506 | 86,935 | 1,519 B | 41 |
| **List (100 records)** | ~3,480 | 287,134 | 82 KB | 1,928 |
| **GetByID** | ~25,800 | 38,785 | 1,163 B | 38 |
| **VerifyChain (1000 records)** | ~409 | 2,445,945 | 926 KB | 28,757 |
| **HashComputation (1KB)** | ~10,085 | 99,158 | 3,576 B | 40 |

### API Endpoints

| Endpoint | Ops/sec | ns/op | Memory/op | Allocs/op |
|----------|---------|-------|-----------|-----------|
| **GET /health** | ~444,640 | 2,249 | 1,648 B | 19 |
| **GET /api/v1/records** (50 records) | ~4,843 | 206,572 | 66 KB | 1,113 |
| **GET /api/v1/records/{id}** | ~18,233 | 54,854 | 2,400 B | 49 |
| **GET /api/v1/verify** (50 records) | ~5,512 | 181,385 | 48 KB | 1,438 |
| **GET /api/v1/snapshot** (100 records) | ~3,181 | 314,389 | 112 KB | 1,954 |
| **JSON Encoding** | ~640,102 | 1,562 | 512 B | 7 |

## Key Insights

### Throughput
- **Single-record appends**: ~12,000 ops/sec (84µs latency)
- **API health checks**: ~445,000 ops/sec (2.2µs latency)
- **Record retrieval by ID**: ~26,000 ops/sec (39µs latency)
- **List queries**: ~3,500 ops/sec (287µs for 100 records)

### Verification Performance
- **Chain verification** scales linearly with record count
- 1,000 records verified in ~2.4ms
- Memory usage: ~930 KB for 1,000 record verification

### Memory Efficiency
- Appending a record: 1.5 KB allocated
- Listing 100 records: 82 KB allocated
- API responses well-optimized (2-66 KB depending on payload size)

### Scalability Notes
- SQLite provides ACID guarantees with minimal overhead
- Parallel writes are serialized by SQLite's locking (expected behavior)
- Read operations are highly concurrent
- API layer adds minimal overhead (~2-20µs per request)

## Running Benchmarks

### All Benchmarks
```bash
go test -bench=. -benchmem ./internal/ledger/... ./internal/api/...
```

### Ledger Only
```bash
go test -bench=. -benchmem ./internal/ledger/...
```

### API Only
```bash
go test -bench=. -benchmem ./internal/api/...
```

### Specific Benchmark
```bash
go test -bench=BenchmarkAppend -benchmem ./internal/ledger/...
```

### With CPU Profiling
```bash
go test -bench=BenchmarkVerifyChain -benchmem -cpuprofile=cpu.prof ./internal/ledger/...
go tool pprof cpu.prof
```

### With Memory Profiling
```bash
go test -bench=BenchmarkList -benchmem -memprofile=mem.prof ./internal/ledger/...
go tool pprof mem.prof
```

## Optimization Opportunities

1. **Batch Appends**: Group multiple records into single transactions
2. **Caching**: Add LRU cache for frequently accessed records
3. **Indexes**: Add indexes on timestamp, type, or source columns
4. **Read Replicas**: Use SQLite WAL mode for concurrent reads
5. **Connection Pooling**: Reuse database connections in API layer

## Production Recommendations

### For High Write Throughput
- Use batch appends (multiple records per transaction)
- Enable SQLite WAL mode: `PRAGMA journal_mode=WAL`
- Consider PostgreSQL backend for distributed writes

### For High Read Throughput
- Enable query result caching
- Use read replicas with SQLite WAL mode
- Add indexes on frequently queried columns

### For Large Datasets
- Regular vacuum operations: `VACUUM`
- Archive old records to separate storage
- Consider partitioning by time period
