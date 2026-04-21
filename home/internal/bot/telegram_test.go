package bot

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestStartSkipsPollingInRelayMode(t *testing.T) {
	svc := NewService(nil, config.BotConfig{
		Enabled:       true,
		Mode:          config.BotModeJWTRelay,
		TelegramToken: "token",
		AllowedChatID: "chat",
	})

	svc.Start()

	if svc.running {
		t.Fatal("expected relay mode to skip polling startup")
	}
}

func TestRelayCommandRejectsUnexpectedChat(t *testing.T) {
	svc := NewService(nil, config.BotConfig{
		Enabled:       true,
		Mode:          config.BotModeJWTRelay,
		TelegramToken: "token",
		AllowedChatID: "chat-1",
	})

	_, err := svc.RelayCommand(context.Background(), "chat-2", "/help")
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected not allowed error, got %v", err)
	}
}

func TestRelayCommandSendsReplyViaTelegram(t *testing.T) {
	var gotChatID string
	var gotText string

	svc := NewService(nil, config.BotConfig{
		Enabled:       true,
		Mode:          config.BotModeJWTRelay,
		TelegramToken: "token",
		AllowedChatID: "chat-1",
	})
	svc.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			values, err := url.ParseQuery(string(body))
			if err != nil {
				return nil, err
			}
			gotChatID = values.Get("chat_id")
			gotText = values.Get("text")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	reply, err := svc.RelayCommand(context.Background(), "", "/help")
	if err != nil {
		t.Fatalf("RelayCommand returned error: %v", err)
	}
	if gotChatID != "chat-1" {
		t.Fatalf("expected chat-1, got %q", gotChatID)
	}
	if gotText == "" || gotText != reply {
		t.Fatalf("expected reply to be sent, got text=%q reply=%q", gotText, reply)
	}
}
