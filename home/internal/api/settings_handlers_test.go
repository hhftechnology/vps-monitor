package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/coolify"
	"github.com/hhftechnology/vps-monitor/internal/services"
)

type fakeCoolifySyncer struct {
	called bool
	err    error
}

func (f *fakeCoolifySyncer) SyncEnvVars(ctx context.Context, resource *coolify.ResourceInfo, envVars map[string]string) error {
	f.called = true
	return f.err
}

func TestSettingsErrorStatusEnvironmentConfigured(t *testing.T) {
	err := fmt.Errorf("update rejected: %w", config.ErrEnvironmentConfigured)
	if got := settingsErrorStatus(err); got != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, got)
	}
}

func TestSettingsErrorStatusDefault(t *testing.T) {
	if got := settingsErrorStatus(errors.New("boom")); got != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, got)
	}
}

func TestApplyCoolifyEnvSyncSkipsDatabaseResources(t *testing.T) {
	syncer := &fakeCoolifySyncer{}
	response := map[string]any{}

	applyCoolifyEnvSync(context.Background(), "host-a", syncer, &coolify.ResourceInfo{
		Type: coolify.ResourceTypeDatabase,
		UUID: "resource-1",
	}, map[string]string{"KEY": "VALUE"}, response)

	if syncer.called {
		t.Fatalf("expected SyncEnvVars not to be called for database resources")
	}
	if got, ok := response["coolify_synced"].(bool); !ok || got {
		t.Fatalf("expected coolify_synced=false, got %#v", response["coolify_synced"])
	}
	if got, ok := response["coolify_error"].(string); !ok || got != "sync not supported for database resources" {
		t.Fatalf("unexpected coolify_error: %#v", response["coolify_error"])
	}
}

func TestApplyCoolifyEnvSyncPropagatesSyncErrors(t *testing.T) {
	syncer := &fakeCoolifySyncer{err: errors.New("upstream failed")}
	response := map[string]any{}

	applyCoolifyEnvSync(context.Background(), "host-a", syncer, &coolify.ResourceInfo{
		Type: coolify.ResourceTypeApplication,
		UUID: "resource-1",
	}, map[string]string{"KEY": "VALUE"}, response)

	if !syncer.called {
		t.Fatalf("expected SyncEnvVars to be called")
	}
	if got, ok := response["coolify_synced"].(bool); !ok || got {
		t.Fatalf("expected coolify_synced=false, got %#v", response["coolify_synced"])
	}
	if got, ok := response["coolify_error"].(string); !ok || got != "upstream failed" {
		t.Fatalf("unexpected coolify_error: %#v", response["coolify_error"])
	}
}

func TestApplyCoolifyEnvSyncSucceeds(t *testing.T) {
	syncer := &fakeCoolifySyncer{}
	response := map[string]any{}

	applyCoolifyEnvSync(context.Background(), "host-a", syncer, &coolify.ResourceInfo{
		Type: coolify.ResourceTypeApplication,
		UUID: "resource-1",
	}, map[string]string{"KEY": "VALUE"}, response)

	if !syncer.called {
		t.Fatalf("expected SyncEnvVars to be called")
	}
	if got, ok := response["coolify_synced"].(bool); !ok || !got {
		t.Fatalf("expected coolify_synced=true, got %#v", response["coolify_synced"])
	}
	if _, hasError := response["coolify_error"]; hasError {
		t.Fatalf("expected no coolify_error key on success, got %#v", response["coolify_error"])
	}
}

func TestApplyCoolifyEnvSyncNilSyncer(t *testing.T) {
	response := map[string]any{}

	applyCoolifyEnvSync(context.Background(), "host-a", nil, &coolify.ResourceInfo{
		Type: coolify.ResourceTypeApplication,
		UUID: "resource-1",
	}, map[string]string{"KEY": "VALUE"}, response)

	if len(response) != 0 {
		t.Fatalf("expected empty response when syncer is nil, got %#v", response)
	}
}

func TestApplyCoolifyEnvSyncNilResource(t *testing.T) {
	syncer := &fakeCoolifySyncer{}
	response := map[string]any{}

	applyCoolifyEnvSync(context.Background(), "host-a", syncer, nil,
		map[string]string{"KEY": "VALUE"}, response)

	if syncer.called {
		t.Fatalf("expected SyncEnvVars not to be called when resource is nil")
	}
	if len(response) != 0 {
		t.Fatalf("expected empty response when resource is nil, got %#v", response)
	}
}

func TestApplyCoolifyEnvSyncServiceResourceType(t *testing.T) {
	syncer := &fakeCoolifySyncer{}
	response := map[string]any{}

	applyCoolifyEnvSync(context.Background(), "host-a", syncer, &coolify.ResourceInfo{
		Type: coolify.ResourceTypeService,
		UUID: "resource-1",
	}, map[string]string{"KEY": "VALUE"}, response)

	if !syncer.called {
		t.Fatalf("expected SyncEnvVars to be called for service resources")
	}
	if got, ok := response["coolify_synced"].(bool); !ok || !got {
		t.Fatalf("expected coolify_synced=true for service resource, got %#v", response["coolify_synced"])
	}
}

func TestSettingsErrorStatusDirectErrEnvironmentConfigured(t *testing.T) {
	if got := settingsErrorStatus(config.ErrEnvironmentConfigured); got != http.StatusConflict {
		t.Fatalf("expected %d for direct ErrEnvironmentConfigured, got %d", http.StatusConflict, got)
	}
}

func TestUpdateBotRejectsIncompleteDiscordConfig(t *testing.T) {
	manager := newTestSettingsManager(t)
	router := &APIRouter{
		manager:  manager,
		registry: services.NewRegistry(nil, nil, nil, manager.Config(), nil),
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/bot", strings.NewReader(`{
		"enabled": false,
		"mode": "polling",
		"telegramToken": "",
		"allowedChatId": "",
		"discord": {
			"enabled": true,
			"botToken": "discord-token",
			"applicationId": "app-1",
			"allowedChannelId": ""
		}
	}`))
	rec := httptest.NewRecorder()

	router.UpdateBot(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d: %s", http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestUpdateBotPreservesMaskedDiscordToken(t *testing.T) {
	manager := newTestSettingsManager(t)
	router := &APIRouter{
		manager:  manager,
		registry: services.NewRegistry(nil, nil, nil, manager.Config(), nil),
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings/bot", strings.NewReader(`{
		"enabled": false,
		"mode": "polling",
		"telegramToken": "",
		"allowedChatId": "",
		"discord": {
			"enabled": true,
			"botToken": "discord-token",
			"applicationId": "app-1",
			"guildId": "guild-1",
			"allowedChannelId": "channel-1"
		}
	}`))
	rec := httptest.NewRecorder()
	router.UpdateBot(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected initial update to succeed, got %d: %s", rec.Code, rec.Body.String())
	}

	router.registry.UpdateConfig(manager.Config())
	req = httptest.NewRequest(http.MethodPut, "/api/v1/settings/bot", strings.NewReader(`{
		"enabled": false,
		"mode": "polling",
		"telegramToken": "",
		"allowedChatId": "",
		"discord": {
			"enabled": true,
			"botToken": "••••••••",
			"applicationId": "app-2",
			"guildId": "",
			"allowedChannelId": "channel-2"
		}
	}`))
	rec = httptest.NewRecorder()
	router.UpdateBot(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected masked update to succeed, got %d: %s", rec.Code, rec.Body.String())
	}

	got := manager.Config().Bot.Discord
	if got.BotToken != "discord-token" {
		t.Fatalf("expected masked update to preserve token, got %q", got.BotToken)
	}
	if got.ApplicationID != "app-2" || got.AllowedChannelID != "channel-2" || got.GuildID != "" {
		t.Fatalf("unexpected discord config after masked update: %+v", got)
	}
}

func TestGetSettingsMasksDiscordToken(t *testing.T) {
	manager := newTestSettingsManager(t)
	enabled := true
	if err := manager.UpdateBotConfig(&config.FileBotConfig{
		Discord: &config.FileDiscordBotConfig{
			Enabled:          &enabled,
			BotToken:         "discord-token",
			ApplicationID:    "app-1",
			AllowedChannelID: "channel-1",
		},
	}); err != nil {
		t.Fatalf("UpdateBotConfig returned error: %v", err)
	}

	router := &APIRouter{
		manager:  manager,
		registry: services.NewRegistry(nil, nil, nil, manager.Config(), nil),
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rec := httptest.NewRecorder()

	router.GetSettings(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var body struct {
		Bot struct {
			Discord struct {
				BotToken string `json:"botToken"`
			} `json:"discord"`
		} `json:"bot"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode settings response: %v", err)
	}
	if body.Bot.Discord.BotToken != secretMask {
		t.Fatalf("expected masked discord token, got %q", body.Bot.Discord.BotToken)
	}
}

func newTestSettingsManager(t *testing.T) *config.Manager {
	t.Helper()
	t.Setenv("CONFIG_PATH", t.TempDir()+"/config.json")
	t.Setenv("BOT_ENABLED", "")
	t.Setenv("BOT_MODE", "")
	t.Setenv("BOT_TELEGRAM_TOKEN", "")
	t.Setenv("BOT_ALLOWED_CHAT_ID", "")
	t.Setenv("BOT_POLL_INTERVAL", "")
	t.Setenv("BOT_DISCORD_ENABLED", "")
	t.Setenv("BOT_DISCORD_TOKEN", "")
	t.Setenv("BOT_DISCORD_APPLICATION_ID", "")
	t.Setenv("BOT_DISCORD_GUILD_ID", "")
	t.Setenv("BOT_DISCORD_ALLOWED_CHANNEL_ID", "")
	return config.NewManager()
}
