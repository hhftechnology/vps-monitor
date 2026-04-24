package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/hhftechnology/vps-monitor/internal/bot"
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

	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}

	reply, err := ar.botService.RelayCommand(r.Context(), req.ChatID, req.Text)
	if err != nil {
		status, message := relayBotCommandErrorResponse(err)
		log.Printf("bot relay command failed: %v", err)
		http.Error(w, message, status)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "Command relayed",
		"reply":   reply,
	})
}

func relayBotCommandErrorResponse(err error) (int, string) {
	switch {
	case errors.Is(err, bot.ErrRelayDisabled):
		return http.StatusConflict, "bot relay mode is disabled"
	case errors.Is(err, bot.ErrRelayNotConfigured):
		return http.StatusConflict, "bot relay is not configured"
	case errors.Is(err, bot.ErrRelayChatNotAllowed):
		return http.StatusForbidden, "chat id is not allowed"
	case errors.Is(err, bot.ErrTelegramSendFailed):
		return http.StatusBadGateway, "failed to send Telegram message"
	default:
		return http.StatusBadRequest, "invalid bot relay command"
	}
}
