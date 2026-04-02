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
	"os"
	"regexp"
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

var safeIdentifierRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type Client struct {
	baseURL    *url.URL
	allowedIPs map[string]struct{}
	apiToken   string
	httpClient *http.Client
}

func newClient(apiURL, apiToken string) (*Client, error) {
	parsedBaseURL, allowedIPs, err := validateAPIURL(apiURL)
	if err != nil {
		return nil, err
	}

	transport, err := newPinnedTransport(parsedBaseURL, allowedIPs)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL:    parsedBaseURL,
		allowedIPs: allowedIPs,
		apiToken:   apiToken,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
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
	_, err := c.doRequest(ctx, http.MethodGet, "/api/v1/version", nil)
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

	// 2. Compute stale env vars to delete after successful upsert
	type staleEnvVar struct {
		key  string
		uuid string
	}
	toDelete := make([]staleEnvVar, 0, len(existing))
	for _, ev := range existing {
		if _, exists := envVars[ev.Key]; !exists {
			toDelete = append(toDelete, staleEnvVar{key: ev.Key, uuid: ev.UUID})
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

	path := fmt.Sprintf("/api/v1/%ss/%s/envs/bulk", resource.Type, resource.UUID)
	if _, err := c.doRequest(ctx, http.MethodPatch, path, body); err != nil {
		return fmt.Errorf("coolify: bulk update failed: %w", err)
	}

	// 4. Delete env vars that no longer exist
	var deletionErrors []string
	for _, stale := range toDelete {
		if delErr := c.deleteEnvVar(ctx, resource, stale.uuid); delErr != nil {
			deletionErrors = append(deletionErrors, fmt.Sprintf("%s (%s): %v", stale.key, stale.uuid, delErr))
		}
	}

	if len(deletionErrors) > 0 {
		return fmt.Errorf("coolify: failed to delete stale env vars: %s", strings.Join(deletionErrors, "; "))
	}

	return nil
}

func (c *Client) listEnvVars(ctx context.Context, resource *ResourceInfo) ([]coolifyEnvVar, error) {
	path := fmt.Sprintf("/api/v1/%ss/%s/envs", resource.Type, resource.UUID)

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
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
	if !safeIdentifierRegex.MatchString(envUUID) {
		return fmt.Errorf("invalid env UUID format")
	}
	if !safeIdentifierRegex.MatchString(resource.UUID) {
		return fmt.Errorf("invalid resource UUID format")
	}
	path := fmt.Sprintf("/api/v1/%ss/%s/envs/%s", resource.Type, resource.UUID, envUUID)
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	requestURL, err := c.resolveRequestURL(path)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	// #nosec G107 -- requestURL is built from a validated base URL and constrained API path.
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bodyReader)
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

func (c *Client) resolveRequestURL(path string) (string, error) {
	if c == nil || c.baseURL == nil {
		return "", fmt.Errorf("coolify client base URL is not configured")
	}
	if !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("invalid Coolify API path")
	}

	baseCopy := *c.baseURL
	requestURL := baseCopy.ResolveReference(&url.URL{Path: path})
	return requestURL.String(), nil
}

func validateAPIURL(raw string) (*url.URL, map[string]struct{}, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid API URL: %w", err)
	}
	allowInsecureHTTP := os.Getenv("COOLIFY_ALLOW_INSECURE_HTTP") == "true"
	if parsed.Scheme != "https" && !(allowInsecureHTTP && parsed.Scheme == "http") {
		return nil, nil, fmt.Errorf("invalid API URL scheme: only https is allowed by default (set COOLIFY_ALLOW_INSECURE_HTTP=true to allow http)")
	}
	hostname := parsed.Hostname()
	if hostname == "" {
		return nil, nil, fmt.Errorf("invalid API URL host")
	}

	allowedIPs := make(map[string]struct{})
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateOrLocalIP(ip) {
			return nil, nil, fmt.Errorf("API URL host is not allowed")
		}
		allowedIPs[ip.String()] = struct{}{}
		return parsed, allowedIPs, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve API URL host: %w", err)
	}
	if len(resolved) == 0 {
		return nil, nil, fmt.Errorf("failed to resolve API URL host")
	}
	for _, addr := range resolved {
		if isPrivateOrLocalIP(addr.IP) {
			return nil, nil, fmt.Errorf("API URL host resolves to a private/local address")
		}
		allowedIPs[addr.IP.String()] = struct{}{}
	}

	return parsed, allowedIPs, nil
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

func newPinnedTransport(baseURL *url.URL, allowedIPs map[string]struct{}) (*http.Transport, error) {
	if baseURL == nil {
		return nil, fmt.Errorf("base URL is required")
	}
	port := baseURL.Port()
	if port == "" {
		if baseURL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	hostname := baseURL.Hostname()
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	return &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			resolved, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve Coolify host %q: %w", hostname, err)
			}

			for _, addr := range resolved {
				ip := addr.IP.String()
				if _, ok := allowedIPs[ip]; !ok {
					continue
				}
				if isPrivateOrLocalIP(addr.IP) {
					continue
				}
				return dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
			}

			return nil, fmt.Errorf("no allowed IP available for Coolify host %q", hostname)
		},
	}, nil
}
