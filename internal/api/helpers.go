package api

import (
	"encoding/json"
	"net/http"
)

// writeJSON encodes v as JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeErr writes an ErrorResponse with the given status and error message.
func writeErr(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, ErrorResponse{Error: err.Error()})
}

// writeErrMsg writes an ErrorResponse with the given status and message string.
func writeErrMsg(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

// readJSON decodes the request body into v.
func readJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
