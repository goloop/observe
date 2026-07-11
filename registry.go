package observe

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Check reports whether a dependency is usable. It should be fast and honour
// the context deadline. A nil error means healthy.
type Check func(ctx context.Context) error

// Registry holds health/readiness checks and build info. It owns the health
// model: liveness (Health) is process-level and always ok; readiness (Ready)
// runs the registered dependency checks. It is safe for concurrent use.
type Registry struct {
	mu     sync.RWMutex
	checks map[string]Check
	order  []string

	service      string
	build        BuildInfo
	checkTimeout time.Duration
	redact       bool
}

// New creates a Registry.
func New(opts ...Option) *Registry {
	r := &Registry{
		checks:       make(map[string]Check),
		checkTimeout: defaultCheckTimeout,
		// Redact check error details by default: a health endpoint is often
		// reachable more widely than intended, and check errors carry
		// hostnames, DSNs or credentials. Opt in to details with
		// WithErrorRedaction(false) for local debugging.
		redact: true,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Check registers or replaces a readiness check. A nil function or an empty
// name is ignored (an empty name would produce a blank JSON key).
func (r *Registry) Check(name string, fn Check) {
	if fn == nil || name == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.checks[name]; !ok {
		r.order = append(r.order, name)
	}
	r.checks[name] = fn
}

// Build returns the configured build info.
func (r *Registry) Build() BuildInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.build
}

// Health returns process-level liveness. It never runs dependency checks and
// is always ok: it answers "is the process up", not "are dependencies usable".
func (r *Registry) Health(context.Context) Status {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return Status{Status: statusOK, Service: r.service, Version: r.build.Version}
}

// Ready runs all registered checks (in parallel, each bounded by the check
// timeout) and returns the aggregated status: error if any check fails.
func (r *Registry) Ready(ctx context.Context) Status {
	results, ok := r.runChecks(ctx)
	s := Status{
		Status:  statusOK,
		Service: r.service,
		Version: r.build.Version,
		Checks:  results,
	}
	if !ok {
		s.Status = statusError
	}
	return s
}

// runChecks executes every check concurrently and collects the results.
func (r *Registry) runChecks(ctx context.Context) (map[string]CheckResult, bool) {
	r.mu.RLock()
	names := append([]string(nil), r.order...)
	checks := make(map[string]Check, len(names))
	for _, n := range names {
		checks[n] = r.checks[n]
	}
	timeout := r.checkTimeout
	redact := r.redact
	r.mu.RUnlock()

	results := make(map[string]CheckResult, len(names))
	if len(names) == 0 {
		return results, true
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	allOK := true

	for _, n := range names {
		wg.Add(1)
		go func(name string, fn Check) {
			defer wg.Done()

			cctx := ctx
			if timeout > 0 {
				var cancel context.CancelFunc
				cctx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}

			start := time.Now()
			err := runOne(cctx, fn)
			res := CheckResult{
				Status:    statusOK,
				LatencyMS: time.Since(start).Milliseconds(),
			}
			if err != nil {
				res.Status = statusError
				if !redact {
					res.Error = err.Error()
				}
			}

			mu.Lock()
			results[name] = res
			if err != nil {
				allOK = false
			}
			mu.Unlock()
		}(n, checks[n])
	}

	wg.Wait()
	return results, allOK
}

// runOne executes a single check, bounded by ctx and isolated from panics. A
// check that hangs past the deadline yields ctx.Err() without blocking the
// handler (the check goroutine leaks, but readiness still answers), and a
// panicking check yields an error instead of crashing the process.
func runOne(ctx context.Context, fn Check) error {
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("check panicked: %v", r)
			}
		}()
		done <- fn(ctx)
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
