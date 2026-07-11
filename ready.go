package observe

import "net/http"

// ReadyHandler serves readiness as JSON. It runs the registered dependency
// checks and returns HTTP 200 when all pass, or HTTP 503 when any fails, so a
// load balancer can stop routing traffic. Mount it at /readyz (or your
// convention).
func (r *Registry) ReadyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		st := r.Ready(req.Context())
		code := http.StatusOK
		if !st.OK() {
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, st)
	})
}
