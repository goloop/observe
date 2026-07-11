package observe

// Status strings used across health and readiness responses.
const (
	statusOK    = "ok"
	statusError = "error"
)

// CheckResult is the outcome of a single dependency check.
type CheckResult struct {
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// OK reports whether the check passed.
func (c CheckResult) OK() bool { return c.Status == statusOK }

// Status is a health or readiness snapshot rendered as JSON.
type Status struct {
	Status  string                 `json:"status"`
	Service string                 `json:"service,omitempty"`
	Version string                 `json:"version,omitempty"`
	Checks  map[string]CheckResult `json:"checks,omitempty"`
}

// OK reports whether the overall status is healthy.
func (s Status) OK() bool { return s.Status == statusOK }
