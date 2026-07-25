package base

import (
	"encoding/json"
	"net/http"
)

// Healthz responds 200 with {"status":"ok"} — shared by all three
// services' /status route. Not called /healthz: Cloud Run's default
// *.run.app domain intercepts that exact path at the platform level and
// serves its own generic 404 before the request ever reaches the
// container, so a route registered under that name is unreachable in
// production even though it works fine locally/through a tunnel.
func Healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
