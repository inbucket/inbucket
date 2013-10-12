package web

import (
	"encoding/json"
	"net/http"
)

func RenderJson(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Expires", "-1")
	enc := json.NewEncoder(w)
	return enc.Encode(data)
}
