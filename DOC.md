# observe - reference

`observe` owns a service's operational-status model: liveness, readiness and
build info. Full English reference; Ukrainian in [DOC.UK.md](DOC.UK.md).

## Contents

- [Model](#model)
- [Construction and options](#construction-and-options)
- [Registering checks](#registering-checks)
- [Health, readiness, build](#health-readiness-build)
- [HTTP handlers](#http-handlers)
- [Response format](#response-format)
- [Decoupling from the lifecycle kernel](#decoupling-from-the-lifecycle-kernel)
- [Concurrency](#concurrency)
- [Scope](#scope)

## Model

A `Registry` holds named readiness checks and build metadata. It distinguishes
two questions:

- **Liveness** (`Health`): is the process up? Always ok; never runs checks.
- **Readiness** (`Ready`): are dependencies usable? Runs every check and fails
  if any fails.

This split matters for orchestration: a failing dependency should take a pod
out of the load-balancer rotation (readiness) without killing the process
(liveness).

## Construction and options

```go
obs := observe.New(opts...)
```

| Option | Effect | Default |
|--------|--------|---------|
| `WithService(name)` | service name in responses | "" |
| `WithBuildInfo(BuildInfo)` | build metadata; sets service if unset | zero |
| `WithCheckTimeout(d)` | per-check deadline in Ready | 2s |
| `WithErrorRedaction(bool)` | hide check error text | false |

`BuildInfo` is `{Service, Version, Commit, BuiltAt}` - populate it via
`-ldflags` or `runtime/debug.ReadBuildInfo`.

## Registering checks

```go
obs.Check("postgres", func(ctx context.Context) error { return pool.Ping(ctx) })
```

A `Check` is `func(context.Context) error`: nil means healthy. Registering the
same name again replaces it. A nil function is ignored. Checks should be fast
and honour the context deadline.

## Health, readiness, build

```go
obs.Health(ctx) Status // liveness, always ok, no checks
obs.Ready(ctx)  Status // runs all checks in parallel, aggregates
obs.Build()     BuildInfo
```

`Ready` runs each check in its own goroutine, each bounded by the check timeout
(the request context still applies). The bound is real: a check that ignores its
context and hangs still yields a timeout error without blocking the response (its
goroutine leaks until it eventually returns, but readiness answers). A panicking
check is recovered and reported as an error rather than crashing the process. The
overall status is `error` if any check fails, `ok` otherwise. An empty registry
is ready.

## HTTP handlers

```go
mux.Handle("/healthz", obs.HealthHandler()) // always 200
mux.Handle("/readyz", obs.ReadyHandler())   // 200 all-ok, else 503
mux.Handle("/buildz", obs.BuildHandler())   // 200, build info
```

Each writes indented JSON with `Content-Type: application/json`.

## Response format

```json
{
  "status": "ok",
  "service": "api",
  "version": "v1.2.3",
  "checks": {
    "postgres": {"status": "ok", "latency_ms": 3},
    "redis": {"status": "error", "latency_ms": 200}
  }
}
```

Every check reports `latency_ms`. `Health`/`Build` omit the `checks` map.
Redaction is **on by default**: a failing check reports
`{"status":"error","latency_ms":N}` without the `error` field, so internal
details (hostnames, DSNs, credentials) never leak from a health endpoint. Pass
`WithErrorRedaction(false)` to include an `"error"` field for local debugging.

## Decoupling from the lifecycle kernel

`observe` never imports a lifecycle kernel and vice versa. Bridge them with a
check over a status snapshot in your wiring code:

```go
obs.Check("app", func(context.Context) error {
	if !a.Status().Healthy() {
		return errors.New("a component failed")
	}
	return nil
})
```

This keeps both packages independent and zero-dependency.

## Concurrency

The `Registry` is safe for concurrent use: `Check` takes a write lock; `Health`,
`Ready` and `Build` take a read lock and run checks on a copied snapshot, so
registering and serving concurrently is safe.

## Scope

`observe` does: health/readiness/build endpoints, dependency checks with
timeout and redaction. A future, optional wave may add a tiny request-metrics
middleware with bounded labels.

`observe` does not: OpenTelemetry, distributed tracing, alerting, dashboards or
log aggregation. It stays compatible with those without depending on them.
