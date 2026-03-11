# Performance

## Benchmark Results (go test -bench)

Run benchmarks with:
```bash
make bench
# or
CGO_ENABLED=1 go test -bench=. -benchmem ./...
```

### Validated Results (Apple M2 Pro, arm64)

| Benchmark | ns/op | B/op | allocs/op |
|-----------|-------|------|-----------|
| BenchmarkDeposit_CleanPass | ~417µs | ~24KB | ~270 |

POST /deposits (clean pass) is well under the 100ms P50 target.

## Target SLAs

| Operation | P50 | P95 | P99 |
|-----------|-----|-----|-----|
| POST /deposits (clean pass) | <100ms | <200ms | <500ms |
| GET /deposits/:id | <10ms | <25ms | <50ms |
| GET /operator/queue | <20ms | <50ms | <100ms |
| POST /settlement/trigger | <500ms | <1s | <2s |

## Bottlenecks

1. **SQLite WAL writes** — serialized writes limit throughput to ~500 TPS single-node
2. **Vendor IQA call** — in this stub it's instant; real vendor adds 200–500ms latency
3. **Image storage** — base64 in SQLite is inefficient; S3 with async upload is preferred

## Optimization Opportunities

- **Connection pooling** — `database/sql` pool already configured via `SetMaxOpenConns`
- **Read replicas** — for GET endpoints, read from replica to reduce write contention
- **Caching** — idempotency keys can be served from Redis instead of SQLite for lower latency
- **Async ledger** — post ledger entries via message queue to decouple from deposit response time
