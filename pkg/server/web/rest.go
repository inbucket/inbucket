package web

import (
	"encoding/json"
	"net/http"
)

// RenderJSON sets the correct HTTP headers for JSON, then writes the specified
// data (typically a struct) encoded in JSON
func RenderJSON(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Expires", "-1")
	enc := json.NewEncoder(w)
	return enc.Encode(data)
}
