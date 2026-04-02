package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

type deviceRegistrationRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// RegisterDevice accepts mobile push device registrations.
//
// Push delivery is not implemented on the server yet, but we accept and validate
// registrations so mobile clients can complete their setup flow without failing.
func (ar *APIRouter) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	var req deviceRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Token) == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}

	WriteJsonResponse(w, http.StatusAccepted, map[string]any{
		"message":   "Device registration accepted",
		"platform":  strings.TrimSpace(req.Platform),
		"delivered": false,
	})
}
