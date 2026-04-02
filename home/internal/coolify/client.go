package coolify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func newClient(apiURL, apiToken string) *Client {
	return &Client{
		apiURL:     apiURL,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// MultiClient routes Coolify API calls to the correct per-host client.
type MultiClient struct {
	clients map[string]*Client
}

// NewMultiClient creates a MultiClient from per-host configs.
// Returns nil if no configs are provided.
func NewMultiClient(hostConfigs []config.CoolifyHostConfig) *MultiClient {
	if len(hostConfigs) == 0 {
		return nil
	}

	clients := make(map[string]*Client, len(hostConfigs))
	for _, hc := range hostConfigs {
		clients[hc.HostName] = newClient(hc.APIURL, hc.APIToken)
	}

	return &MultiClient{clients: clients}
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
func NewSingleClient(apiURL, apiToken string) *Client {
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
	for _, ev := range existing {
		if _, exists := envVars[ev.Key]; !exists {
			if delErr := c.deleteEnvVar(ctx, resource, ev.UUID); delErr != nil {
				log.Printf("Warning: failed to delete Coolify env var %s (%s): %v", ev.Key, ev.UUID, delErr)
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return respBody, nil
}
