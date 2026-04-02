package coolify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/config"
)

func TestValidateAPIURLRejectsPrivateIP(t *testing.T) {
	if err := validateAPIURL("http://127.0.0.1"); err == nil {
		t.Fatalf("expected private loopback URL to be rejected")
	}
}

func TestValidateAPIURLAllowsPublicIP(t *testing.T) {
	if err := validateAPIURL("https://8.8.8.8"); err != nil {
		t.Fatalf("expected public IP URL to be allowed, got %v", err)
	}
}

func TestNewMultiClientRejectsDuplicateHostNames(t *testing.T) {
	_, err := NewMultiClient([]config.CoolifyHostConfig{
		{HostName: "host-a", APIURL: "https://8.8.8.8", APIToken: "a"},
		{HostName: "host-a", APIURL: "https://1.1.1.1", APIToken: "b"},
	})
	if err == nil {
		t.Fatalf("expected duplicate host names to return an error")
	}
}

func TestSyncEnvVarsReturnsDeletionErrors(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/applications/resource-1/envs":
			body, _ := json.Marshal([]map[string]string{
				{"uuid": "uuid-old", "key": "OLD_KEY"},
				{"uuid": "uuid-keep", "key": "KEEP_KEY"},
			})
			return response(http.StatusOK, string(body)), nil
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/applications/resource-1/envs/uuid-old":
			return response(http.StatusInternalServerError, "delete failed"), nil
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/applications/resource-1/envs/bulk":
			return response(http.StatusOK, `{}`), nil
		default:
			return response(http.StatusNotFound, "not found"), nil
		}
	})

	client := &Client{
		apiURL:     "https://example.com",
		apiToken:   "token",
		httpClient: &http.Client{Transport: transport},
	}
	resource := &ResourceInfo{Type: ResourceTypeApplication, UUID: "resource-1"}
	err := client.SyncEnvVars(context.Background(), resource, map[string]string{
		"KEEP_KEY": "value",
		"NEW_KEY":  "new",
	})
	if err == nil {
		t.Fatalf("expected stale delete failure to propagate")
	}
	if !strings.Contains(err.Error(), "OLD_KEY (uuid-old)") {
		t.Fatalf("expected error to include stale key/uuid context, got %v", err)
	}
	if !strings.Contains(err.Error(), "delete failed") {
		t.Fatalf("expected error to include response body snippet, got %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
