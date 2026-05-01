// Package api implements the JSON HTTP API consumed by the React PWA.
package api

import (
	"encoding/json"
	"net/http"
)

// maxRequestBody caps decoded JSON payloads. The largest legitimate request is
// a create/update expense, whose fields are a single float and three short
// strings — well under 1 KiB. 64 KiB leaves slack for verbose descriptions
// without leaving the door open to a multi-megabyte memory exhaustion via
// json.Decode on an authenticated endpoint.
const maxRequestBody = 64 * 1024

type errorEnvelope struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorEnvelope{Error: msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	return json.NewDecoder(r.Body).Decode(v)
}
