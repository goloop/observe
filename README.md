[![Go Reference](https://img.shields.io/badge/godoc-reference-blue)](https://pkg.go.dev/github.com/goloop/observe) [![License](https://img.shields.io/badge/license-MIT-brightgreen)](https://github.com/goloop/observe/blob/master/LICENSE) [![Stay with Ukraine](https://img.shields.io/static/v1?label=Stay%20with&message=Ukraine%20♥&color=ffD700&labelColor=0057B8&style=flat)](https://u24.gov.ua/)

# observe

`observe` gives every Go service the same operational-status endpoints -
liveness, readiness and build info - without copy-pasted handlers. Zero
dependencies, standard library only.

It is deliberately **not** a metrics/tracing platform: no Prometheus client, no
OpenTelemetry SDK. It stays compatible with those tools without depending on
them.

## Install

```bash
go get github.com/goloop/observe
```

## Quick start

```go
obs := observe.New(
	observe.WithBuildInfo(observe.BuildInfo{Service: "api", Version: "v1.2.3"}),
	observe.WithErrorRedaction(true), // hide details in production
)

obs.Check("postgres", func(ctx context.Context) error { return pool.Ping(ctx) })
obs.Check("redis", func(ctx context.Context) error { return rdb.Ping(ctx) })

mux.Handle("/healthz", obs.HealthHandler()) // 200 while the process lives
mux.Handle("/readyz", obs.ReadyHandler())   // 503 if a dependency is down
mux.Handle("/buildz", obs.BuildHandler())
```

## Health vs readiness

- **Health** (`/healthz`) is process-level liveness: "is the process up". It
  never runs dependency checks and is always `200`.
- **Readiness** (`/readyz`) runs the registered checks in parallel (each bounded
  by a timeout, even if a check ignores its context; a panicking check is
  recovered) and returns `503` when any dependency is down, so a load balancer
  stops routing traffic.

Response shape:

```json
{
  "status": "error",
  "service": "api",
  "version": "v1.2.3",
  "checks": {
    "postgres": {"status": "ok", "latency_ms": 3},
    "redis": {"status": "error", "latency_ms": 200}
  }
}
```

Every check reports `latency_ms`. Error details are **redacted by default**, so
the failing status is reported without leaking the underlying error (which can
carry hostnames, DSNs or credentials). Pass `WithErrorRedaction(false)` to add
an `"error"` field for local debugging - not in production.

## Decoupling

`observe` never imports a lifecycle kernel. To surface application lifecycle
state, register it as an ordinary check over a status snapshot:

```go
obs.Check("app", func(context.Context) error {
	if !a.Status().Healthy() {
		return errors.New("a component failed")
	}
	return nil
})
```

## Documentation

- English reference: [DOC.md](DOC.md)
- Ukrainian reference: [DOC.UK.md](DOC.UK.md)

## License

MIT - see [LICENSE](LICENSE).
