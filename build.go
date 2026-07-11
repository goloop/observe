package observe

// BuildInfo identifies the running build. It is surfaced in health and
// readiness responses and by BuildHandler. Populate it at build time with
// -ldflags, or from runtime/debug.ReadBuildInfo.
type BuildInfo struct {
	Service string `json:"service,omitempty"`
	Version string `json:"version,omitempty"`
	Commit  string `json:"commit,omitempty"`
	BuiltAt string `json:"built_at,omitempty"`
}
