package coolify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/config"
)

type ResourceType string

const (
	ResourceTypeApplication ResourceType = "application"
	ResourceTypeService     ResourceType = "service"
	ResourceTypeDatabase    ResourceType = "database"
)

type ResourceInfo struct {
	Type ResourceType
	UUID string
}

type Client struct {
	apiURL     string
	apiToken   string
	httpClient *http.Client
}

func newClient(apiURL, apiToken string) (*Client, error) {
	if err := validateAPIURL(apiURL); err != nil {
		return nil, err
	}

	return &Client{
		apiURL:     apiURL,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// MultiClient routes Coolify API calls to the correct per-host client.
type MultiClient struct {
	clients map[string]*Client
}

// NewMultiClient creates a MultiClient from per-host configs.
// Returns nil if no configs are provided.
func NewMultiClient(hostConfigs []config.CoolifyHostConfig) (*MultiClient, error) {
	if len(hostConfigs) == 0 {
		return nil, nil
	}

	clients := make(map[string]*Client, len(hostConfigs))
	for _, hc := range hostConfigs {
		if _, exists := clients[hc.HostName]; exists {
			return nil, fmt.Errorf("duplicate Coolify host name: %s", hc.HostName)
		}

		client, err := newClient(hc.APIURL, hc.APIToken)
		if err != nil {
			return nil, fmt.Errorf("invalid Coolify host %s: %w", hc.HostName, err)
		}
		clients[hc.HostName] = client
	}

	return &MultiClient{clients: clients}, nil
}

// GetClient returns the Coolify client for the given Docker host name.
// Returns nil if no config exists for that host.
func (mc *MultiClient) GetClient(hostName string) *Client {
	if mc == nil {
		return nil
	}
	return mc.clients[hostName]
}

// TestConnection checks if the Coolify API is reachable by calling /api/v1/version.
func (c *Client) TestConnection(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("coolify client is nil")
	}
	url := fmt.Sprintf("%s/api/v1/version", c.apiURL)
	_, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("coolify API unreachable: %w", err)
	}
	return nil
}

// NewSingleClient creates a single Coolify client for testing connections.
func NewSingleClient(apiURL, apiToken string) (*Client, error) {
	return newClient(apiURL, apiToken)
}

// ExtractResourceInfo checks container labels for Coolify management info.
// Returns nil if the container is not managed by Coolify.
func ExtractResourceInfo(labels map[string]string) *ResourceInfo {
	if labels["coolify.managed"] != "true" {
		return nil
	}

	uuid := labels["com.docker.compose.project"]
	if uuid == "" {
		return nil
	}

	switch labels["coolify.type"] {
	case "application":
		return &ResourceInfo{Type: ResourceTypeApplication, UUID: uuid}
	case "service":
		return &ResourceInfo{Type: ResourceTypeService, UUID: uuid}
	case "database":
		return &ResourceInfo{Type: ResourceTypeDatabase, UUID: uuid}
	default:
		return &ResourceInfo{Type: ResourceTypeApplication, UUID: uuid}
	}
}

type envVarEntry struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	IsPreview bool   `json:"is_preview"`
}

type bulkEnvPayload struct {
	Data []envVarEntry `json:"data"`
}

type coolifyEnvVar struct {
	UUID string `json:"uuid"`
	Key  string `json:"key"`
}

// SyncEnvVars syncs environment variables to Coolify's API so they persist
// across redeployments. It upserts new/changed vars and deletes removed ones.
func (c *Client) SyncEnvVars(ctx context.Context, resource *ResourceInfo, envVars map[string]string) error {
	if c == nil || resource == nil {
		return nil
	}

	if resource.Type == ResourceTypeDatabase {
		return fmt.Errorf("coolify: syncing env vars for database resources is not supported")
	}

	// 1. Fetch current env vars from Coolify to find deletions
	existing, err := c.listEnvVars(ctx, resource)
	if err != nil {
		return fmt.Errorf("coolify: failed to fetch existing env vars: %w", err)
	}

	// 2. Delete env vars that no longer exist
	var deletionErrors []string
	for _, ev := range existing {
		if _, exists := envVars[ev.Key]; !exists {
			if delErr := c.deleteEnvVar(ctx, resource, ev.UUID); delErr != nil {
				deletionErrors = append(deletionErrors, fmt.Sprintf("%s (%s): %v", ev.Key, ev.UUID, delErr))
			}
		}
	}

	// 3. Bulk upsert the current env vars
	entries := make([]envVarEntry, 0, len(envVars))
	for key, value := range envVars {
		entries = append(entries, envVarEntry{Key: key, Value: value, IsPreview: false})
	}

	body, err := json.Marshal(bulkEnvPayload{Data: entries})
	if err != nil {
		return fmt.Errorf("coolify: failed to marshal env vars: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/%ss/%s/envs/bulk", c.apiURL, resource.Type, resource.UUID)
	if _, err := c.doRequest(ctx, http.MethodPatch, url, body); err != nil {
		return fmt.Errorf("coolify: bulk update failed: %w", err)
	}

	if len(deletionErrors) > 0 {
		return fmt.Errorf("coolify: failed to delete stale env vars: %s", strings.Join(deletionErrors, "; "))
	}

	return nil
}

func (c *Client) listEnvVars(ctx context.Context, resource *ResourceInfo) ([]coolifyEnvVar, error) {
	url := fmt.Sprintf("%s/api/v1/%ss/%s/envs", c.apiURL, resource.Type, resource.UUID)

	respBody, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var envVars []coolifyEnvVar
	if err := json.Unmarshal(respBody, &envVars); err != nil {
		return nil, err
	}
	return envVars, nil
}

func (c *Client) deleteEnvVar(ctx context.Context, resource *ResourceInfo, envUUID string) error {
	url := fmt.Sprintf("%s/api/v1/%ss/%s/envs/%s", c.apiURL, resource.Type, resource.UUID, envUUID)
	_, err := c.doRequest(ctx, http.MethodDelete, url, nil)
	return err
}

func (c *Client) doRequest(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippetBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 512))
		if readErr != nil {
			return nil, fmt.Errorf("API returned status %d (failed to read response body: %w)", resp.StatusCode, readErr)
		}
		snippet := strings.TrimSpace(strings.ReplaceAll(string(snippetBytes), "\n", " "))
		if snippet == "" {
			snippet = "empty response"
		}
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, snippet)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respBody, nil
}

func validateAPIURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid API URL scheme: %s", parsed.Scheme)
	}
	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("invalid API URL host")
	}

	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateOrLocalIP(ip) {
			return fmt.Errorf("API URL host is not allowed")
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve API URL host: %w", err)
	}
	if len(resolved) == 0 {
		return fmt.Errorf("failed to resolve API URL host")
	}
	for _, addr := range resolved {
		if isPrivateOrLocalIP(addr.IP) {
			return fmt.Errorf("API URL host resolves to a private/local address")
		}
	}

	return nil
}

func isPrivateOrLocalIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return true
	}
	return ip.IsPrivate()
}
