package bot

import (
	"context"
	"encoding/json"
	"errors"
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
	if !errors.Is(err, ErrRelayChatNotAllowed) {
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

func TestPollOnceUsesCancellableContext(t *testing.T) {
	svc := NewService(nil, config.BotConfig{
		Enabled:       true,
		Mode:          config.BotModePolling,
		TelegramToken: "token",
		AllowedChatID: "chat-1",
	})
	svc.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			<-req.Context().Done()
			return nil, req.Context().Err()
		}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := svc.pollOnce(ctx, svc.cfg)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestSharedCommandHandlerKeepsTelegramHelpReply(t *testing.T) {
	reply := newCommandHandler(nil).handle("/help")
	if !strings.Contains(reply, "/status") || !strings.Contains(reply, "/critical") {
		t.Fatalf("expected help reply to list existing commands, got %q", reply)
	}
}

func TestDiscordInteractionReplyRejectsUnexpectedChannel(t *testing.T) {
	svc := NewService(nil, config.BotConfig{})
	var interaction discordInteraction
	interaction.Type = discordInteractionApplicationCommand
	interaction.ChannelID = "channel-2"
	interaction.Data.Name = "help"

	reply := svc.discordInteractionReply(config.DiscordBotConfig{
		Enabled:          true,
		BotToken:         "token",
		ApplicationID:    "app",
		AllowedChannelID: "channel-1",
	}, interaction)
	if !strings.Contains(reply, "channel is not allowed") {
		t.Fatalf("expected channel rejection, got %q", reply)
	}
}

func TestSendDiscordTestMessagePostsToChannel(t *testing.T) {
	var gotAuth string
	var gotPath string
	var gotContent string

	svc := NewService(nil, config.BotConfig{})
	svc.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotAuth = req.Header.Get("Authorization")
			gotPath = req.URL.Path
			var body struct {
				Content string `json:"content"`
			}
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				return nil, err
			}
			gotContent = body.Content
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"id":"message-1"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}
	svc.discordAPIBase = "https://discord.test"

	if err := svc.SendDiscordTestMessage(context.Background(), "token-1", "channel-1"); err != nil {
		t.Fatalf("SendDiscordTestMessage returned error: %v", err)
	}
	if gotAuth != "Bot token-1" {
		t.Fatalf("unexpected auth header %q", gotAuth)
	}
	if gotPath != "/channels/channel-1/messages" {
		t.Fatalf("unexpected path %q", gotPath)
	}
	if !strings.Contains(gotContent, "VPS Monitor Discord bot test successful") {
		t.Fatalf("unexpected content %q", gotContent)
	}
}
