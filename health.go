package observe

import (
	"encoding/json"
	"net/http"
)

// HealthHandler serves liveness as JSON with HTTP 200. It does not run
// dependency checks: it answers whether the process is up. Mount it at
// /healthz (or your convention).
func (r *Registry) HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusOK, r.Health(req.Context()))
	})
}

// BuildHandler serves the build info as JSON with HTTP 200.
func (r *Registry) BuildHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, r.Build())
	})
}

// writeJSON writes v as an indented JSON response with the given status code.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
