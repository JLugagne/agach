package client_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func writeJSON(t *testing.T, w http.ResponseWriter, status int, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Errorf("writeJSON: %v", err)
	}
}

func successResponse(data any) map[string]any {
	return map[string]any{
		"status": "success",
		"data":   data,
	}
}
