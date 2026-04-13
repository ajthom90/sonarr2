package v6

import (
	"encoding/json"
	"net/http"
)

// ProblemDetail implements RFC 9457 (Problem Details for HTTP APIs).
type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// WriteError writes an RFC 9457 problem detail response.
func WriteError(w http.ResponseWriter, r *http.Request, status int, detail string) {
	pd := ProblemDetail{
		Type:     "about:blank",
		Title:    http.StatusText(status),
		Status:   status,
		Detail:   detail,
		Instance: r.URL.Path,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(pd)
}

// WriteNotFound writes a 404 Not Found problem detail.
func WriteNotFound(w http.ResponseWriter, r *http.Request, detail string) {
	WriteError(w, r, http.StatusNotFound, detail)
}

// WriteBadRequest writes a 400 Bad Request problem detail.
func WriteBadRequest(w http.ResponseWriter, r *http.Request, detail string) {
	WriteError(w, r, http.StatusBadRequest, detail)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
