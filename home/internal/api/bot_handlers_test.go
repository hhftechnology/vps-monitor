package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/auth"
	"github.com/hhftechnology/vps-monitor/internal/bot"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/services"
)

type fakeBotRelayService struct {
	reply     string
	err       error
	gotText   string
	gotChat   string
	callCount int
}

func (f *fakeBotRelayService) RelayCommand(_ context.Context, chatID, text string) (string, error) {
	f.callCount++
	f.gotChat = chatID
	f.gotText = text
	return f.reply, f.err
}

func TestRelayBotCommandRejectsWhenAuthDisabled(t *testing.T) {
	router := &APIRouter{
		registry: services.NewRegistry(nil, nil, auth.NewDisabledService(), &config.Config{
			Bot: config.BotConfig{Enabled: true, Mode: config.BotModeJWTRelay},
		}, nil),
		botService: &fakeBotRelayService{},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bot/relay/command", strings.NewReader(`{"text":"/help"}`))
	rec := httptest.NewRecorder()
	router.RelayBotCommand(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestRelayBotCommandReturnsReply(t *testing.T) {
	relay := &fakeBotRelayService{reply: "ok"}
	router := &APIRouter{
		registry: services.NewRegistry(nil, nil, newUsableAuthService(t), &config.Config{
			Bot: config.BotConfig{Enabled: true, Mode: config.BotModeJWTRelay},
		}, nil),
		botService: relay,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bot/relay/command", strings.NewReader(`{"text":"  /help  ","chatId":"123"}`))
	rec := httptest.NewRecorder()
	router.RelayBotCommand(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content type, got %q", ct)
	}
	var body struct {
		Message string `json:"message"`
		Reply   string `json:"reply"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Message != "Command relayed" || body.Reply != "ok" {
		t.Fatalf("unexpected response body: %+v", body)
	}
	if relay.gotText != "/help" || relay.gotChat != "123" {
		t.Fatalf("expected trimmed command and chat id, got text=%q chat=%q", relay.gotText, relay.gotChat)
	}
}

func TestRelayBotCommandRejectsNilBotService(t *testing.T) {
	router := &APIRouter{}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bot/relay/command", strings.NewReader(`{"text":"/help"}`))
	rec := httptest.NewRecorder()
	router.RelayBotCommand(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}
}

func TestRelayBotCommandRejectsWhenModeIsNotRelay(t *testing.T) {
	router := &APIRouter{
		registry: services.NewRegistry(nil, nil, newUsableAuthService(t), &config.Config{
			Bot: config.BotConfig{Enabled: true, Mode: config.BotModePolling},
		}, nil),
		botService: &fakeBotRelayService{},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bot/relay/command", strings.NewReader(`{"text":"/help"}`))
	rec := httptest.NewRecorder()
	router.RelayBotCommand(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestRelayBotCommandRejectsInvalidRequestBody(t *testing.T) {
	router := newRelayBotCommandTestRouter(t, &fakeBotRelayService{})

	for _, body := range []string{`{`, `{"text":""}`, `{"text":"   "}`} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/bot/relay/command", strings.NewReader(body))
		rec := httptest.NewRecorder()
		router.RelayBotCommand(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected %d for body %q, got %d", http.StatusBadRequest, body, rec.Code)
		}
	}
}

func TestRelayBotCommandMapsRelayErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
		body string
	}{
		{name: "unknown", err: errors.New("internal detail"), want: http.StatusBadRequest, body: "invalid bot relay command"},
		{name: "disabled", err: bot.ErrRelayDisabled, want: http.StatusConflict, body: "bot relay mode is disabled"},
		{name: "not configured", err: bot.ErrRelayNotConfigured, want: http.StatusConflict, body: "bot relay is not configured"},
		{name: "chat not allowed", err: bot.ErrRelayChatNotAllowed, want: http.StatusForbidden, body: "chat id is not allowed"},
		{name: "send failed", err: bot.ErrTelegramSendFailed, want: http.StatusBadGateway, body: "failed to send Telegram message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newRelayBotCommandTestRouter(t, &fakeBotRelayService{err: tt.err})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/bot/relay/command", strings.NewReader(`{"text":"/help"}`))
			rec := httptest.NewRecorder()
			router.RelayBotCommand(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("expected %d, got %d", tt.want, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.body) {
				t.Fatalf("expected sanitized body %q, got %q", tt.body, rec.Body.String())
			}
		})
	}
}

func newRelayBotCommandTestRouter(t *testing.T, relay *fakeBotRelayService) *APIRouter {
	t.Helper()
	return &APIRouter{
		registry: services.NewRegistry(nil, nil, newUsableAuthService(t), &config.Config{
			Bot: config.BotConfig{Enabled: true, Mode: config.BotModeJWTRelay},
		}, nil),
		botService: relay,
	}
}

func newUsableAuthService(t *testing.T) *auth.Service {
	t.Helper()
	hash, err := auth.HashPassword("secret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	return auth.NewServiceFromFileConfig(&config.FileAuthConfig{
		Enabled:           true,
		JWTSecret:         "jwt-secret",
		AdminUsername:     "admin",
		AdminPasswordHash: hash,
	})
}
