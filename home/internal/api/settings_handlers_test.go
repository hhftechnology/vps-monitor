package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/coolify"
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