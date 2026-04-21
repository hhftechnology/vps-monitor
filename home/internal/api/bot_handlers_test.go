package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/auth"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/services"
)

type fakeBotRelayService struct {
	reply string
	err   error
}

func (f *fakeBotRelayService) RelayCommand(_ context.Context, _, _ string) (string, error) {
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
	router := &APIRouter{
		registry: services.NewRegistry(nil, nil, &auth.Service{}, &config.Config{
			Bot: config.BotConfig{Enabled: true, Mode: config.BotModeJWTRelay},
		}, nil),
		botService: &fakeBotRelayService{reply: "ok"},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bot/relay/command", strings.NewReader(`{"text":"/help","chatId":"123"}`))
	rec := httptest.NewRecorder()
	router.RelayBotCommand(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"reply":"ok"`) {
		t.Fatalf("expected reply in body, got %s", rec.Body.String())
	}
}
