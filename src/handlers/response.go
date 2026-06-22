package handlers

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeSuccess(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, map[string]any{
		"status": "success",
		"data":   data,
	})
}

func writeSuccessWithMessage(w http.ResponseWriter, status int, message string, data any) {
	writeJSON(w, status, map[string]any{
		"status":  "success",
		"message": message,
		"data":    data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"status":  "error",
		"message": message,
	})
}