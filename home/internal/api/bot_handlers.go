package api

import (
	"encoding/json"
	"net/http"

	"github.com/hhftechnology/vps-monitor/internal/config"
)

func (ar *APIRouter) RelayBotCommand(w http.ResponseWriter, r *http.Request) {
	if ar.botService == nil {
		http.Error(w, "bot service unavailable", http.StatusServiceUnavailable)
		return
	}
	if svc := ar.registry.Auth(); svc == nil || svc.IsDisabled() {
		http.Error(w, "auth must be enabled before using bot relay mode", http.StatusConflict)
		return
	}

	if ar.registry.Config().Bot.Mode != config.BotModeJWTRelay {
		http.Error(w, "bot relay mode is disabled", http.StatusConflict)
		return
	}

	var req struct {
		Text   string `json:"text"`
		ChatID string `json:"chatId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}

	reply, err := ar.botService.RelayCommand(r.Context(), req.ChatID, req.Text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "Command relayed",
		"reply":   reply,
	})
}
