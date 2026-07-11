package observe

import "time"

// defaultCheckTimeout bounds a single readiness check unless overridden.
const defaultCheckTimeout = 2 * time.Second

// Option configures a Registry at construction time.
type Option func(*Registry)

// WithService sets the service name reported in responses.
func WithService(name string) Option {
	return func(r *Registry) { r.service = name }
}

// WithBuildInfo sets the build metadata. If the service name is not set
// separately, it is taken from the build info.
func WithBuildInfo(b BuildInfo) Option {
	return func(r *Registry) {
		r.build = b
		if r.service == "" {
			r.service = b.Service
		}
	}
}

// WithCheckTimeout sets the per-check deadline used by Ready. A non-positive
// value disables the per-check timeout (the request context still applies).
func WithCheckTimeout(d time.Duration) Option {
	return func(r *Registry) { r.checkTimeout = d }
}

// WithErrorRedaction controls whether check error details appear in responses;
// the failing status is always reported either way. Redaction is ON by default
// so internals never leak by accident. Pass false to expose details for local
// debugging - not in production.
func WithErrorRedaction(redact bool) Option {
	return func(r *Registry) { r.redact = redact }
}
