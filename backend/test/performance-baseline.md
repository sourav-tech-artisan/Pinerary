# Backend performance baseline

Run the deterministic CPU benchmark on the deployment target:

```bash
go test -run '^$' -bench BenchmarkCleanTenThousandPoints -benchmem ./internal/tracking
```

Record the Go version, CPU, `ns/op`, `B/op`, and `allocs/op` with a release tag. Treat a
repeatable regression above 20% as a review trigger, not an automatic architecture change.

For API load tests, use anonymized synthetic data and measure GPS batch ingestion (500
points), nearby search (50 PostGIS candidates and three Valhalla matrices), and upload-intent
creation separately. Provider latency must be reported apart from API/database latency.
