package coolify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/hhftechnology/vps-monitor/internal/config"
)

func TestValidateAPIURLRejectsPrivateIP(t *testing.T) {
	if _, _, err := validateAPIURL("https://127.0.0.1"); err == nil {
		t.Fatalf("expected private loopback URL to be rejected")
	}
}

func TestValidateAPIURLAllowsPublicIP(t *testing.T) {
	if _, _, err := validateAPIURL("https://8.8.8.8"); err != nil {
		t.Fatalf("expected public IP URL to be allowed, got %v", err)
	}
}

func TestValidateAPIURLRejectsPrivateRanges(t *testing.T) {
	for _, raw := range []string{
		"https://10.0.0.1",
		"https://192.168.1.1",
		"https://172.16.0.1",
	} {
		if _, _, err := validateAPIURL(raw); err == nil {
			t.Fatalf("expected private range URL %q to be rejected", raw)
		}
	}
}

func TestValidateAPIURLRejectsIPv6Loopback(t *testing.T) {
	if _, _, err := validateAPIURL("https://[::1]"); err == nil {
		t.Fatalf("expected IPv6 loopback URL to be rejected")
	}
}

func TestValidateAPIURLRejectsInvalidURLs(t *testing.T) {
	for _, raw := range []string{
		"http:////",
		"://nohost",
	} {
		if _, _, err := validateAPIURL(raw); err == nil {
			t.Fatalf("expected malformed URL %q to be rejected", raw)
		}
	}
}

func TestValidateAPIURLRejectsHTTPWithoutOptIn(t *testing.T) {
	t.Setenv("COOLIFY_ALLOW_INSECURE_HTTP", "")
	if _, _, err := validateAPIURL("http://8.8.8.8"); err == nil {
		t.Fatalf("expected http URL to be rejected without opt-in")
	}
}

func TestValidateAPIURLAllowsHTTPWithOptIn(t *testing.T) {
	t.Setenv("COOLIFY_ALLOW_INSECURE_HTTP", "true")
	if _, _, err := validateAPIURL("http://8.8.8.8"); err != nil {
		t.Fatalf("expected http URL to be allowed with opt-in, got %v", err)
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
		baseURL:    mustParseURL(t, "https://example.com"),
		allowedIPs: map[string]struct{}{"127.0.0.1": {}},
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

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse URL %q: %v", raw, err)
	}
	return parsed
}
