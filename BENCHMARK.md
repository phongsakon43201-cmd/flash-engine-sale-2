# Flash Sale Engine benchmark guide

This file intentionally separates correctness evidence from capacity claims. The repository does not claim a fixed RPS or latency result because those numbers depend on the host, Docker resources, network, dataset, and test duration.

## 1. Correctness and race checks

Run:

```bash
go test ./...
go test -race ./...
```

`TestCreateFlashSaleOrder_HighConcurrency_ZeroOverselling` launches 100 goroutines against a thread-safe in-memory cache with an initial stock of 10. It verifies that exactly 10 requests reserve inventory and that the remaining stock is zero. This is a business-logic concurrency test; it is not an end-to-end Redis, SQS, or MongoDB throughput benchmark.

Worker tests also verify that a transient MongoDB transaction failure can be retried and that an already-committed `order_id` is handled idempotently.

## 2. End-to-end k6 test

Start the local stack and create or select a flash-sale product:

```bash
docker compose up --build -d
k6 run -e PRODUCT_ID=<mongo-object-id> scripts/load_test.js
```

The script ramps to 2,000 virtual users. HTTP 202, expected sold-out HTTP 400 responses, and rate-limit HTTP 429 responses are classified separately from transport failures.

Before publishing results, record:

- CPU, memory, operating system, Docker resource limits, and k6 version
- test start time, duration, initial inventory, and product ID
- raw k6 summary output
- API, worker, Redis, MongoDB, and SQS/LocalStack resource utilization
- accepted, sold-out, rate-limited, DLQ, and transport-error counts
- p50, p95, and p99 latency for accepted requests separately from rejections

## 3. Acceptance criteria

The default script checks:

- p95 HTTP duration below 100 ms
- transport failure rate below 1%
- every response is accepted, sold out, or rate limited
- individual response duration below 50 ms for the stricter check counter

These are targets, not pre-recorded results. Save raw output before describing a target as achieved.

## Monitoring

- Prometheus metrics: `http://localhost:8080/metrics`
- Prometheus UI: `http://localhost:9090`
- Grafana: `http://localhost:3000` (`admin` / `admin`, local development only)
