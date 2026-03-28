package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterDevice(t *testing.T) {
	t.Run("accepts valid registration", func(t *testing.T) {
		router := &APIRouter{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/register", bytes.NewBufferString(`{"token":"abc","platform":"ios"}`))
		rec := httptest.NewRecorder()

		router.RegisterDevice(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if body["message"] != "Device registration accepted" {
			t.Fatalf("unexpected message: %v", body["message"])
		}
	})

	t.Run("rejects missing token", func(t *testing.T) {
		router := &APIRouter{}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/register", bytes.NewBufferString(`{"platform":"ios"}`))
		rec := httptest.NewRecorder()

		router.RegisterDevice(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})
}
