// Package observe provides health, readiness and build-info endpoints for Go
// services, standard library only.
//
// It owns the operational-status model so every service exposes the same
// /healthz, /readyz and build-info shape without copy-pasted handlers. It is
// deliberately not a metrics/tracing platform: no Prometheus client, no
// OpenTelemetry SDK.
//
// # Health vs readiness
//
// Health is process-level liveness: it answers "is the process up" and is
// always ok. Readiness runs the registered dependency checks (database, cache,
// queue, ...) and fails when any dependency is unusable, so a load balancer can
// stop routing traffic.
//
//	obs := observe.New(
//	    observe.WithBuildInfo(observe.BuildInfo{Service: "api", Version: "v1.2.3"}),
//	    // Error details are redacted by default; opt in with
//	    // observe.WithErrorRedaction(false) for local debugging.
//	)
//	obs.Check("postgres", func(ctx context.Context) error { return pool.Ping(ctx) })
//	obs.Check("redis", func(ctx context.Context) error { return rdb.Ping(ctx) })
//
//	mux.Handle("/healthz", obs.HealthHandler()) // 200 while the process lives
//	mux.Handle("/readyz", obs.ReadyHandler())   // 503 if a dependency is down
//	mux.Handle("/buildz", obs.BuildHandler())
//
// # Decoupling
//
// observe never imports the lifecycle kernel. To surface application lifecycle
// state, register it as an ordinary check over the kernel's status snapshot:
//
//	obs.Check("app", func(context.Context) error {
//	    if !a.Status().Healthy() {
//	        return errors.New("a component failed")
//	    }
//	    return nil
//	})
//
// See DOC.md (English) and DOC.UK.md (Ukrainian) for the full reference.
package observe
