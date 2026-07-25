package base

import (
	"encoding/json"
	"net/http"
)

// Healthz responds 200 with {"status":"ok"} — shared by all three
// services' /healthz route.
func Healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
