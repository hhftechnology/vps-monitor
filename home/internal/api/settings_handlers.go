package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/auth"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/coolify"
	"github.com/hhftechnology/vps-monitor/internal/docker"
)

const secretMask = "••••••••"

var hostNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// GetSettings returns the current configuration with source tracking and masked secrets.
func (ar *APIRouter) GetSettings(w http.ResponseWriter, r *http.Request) {
	sources := ar.manager.Sources()
	cfg := ar.registry.Config()
	fc := ar.manager.FileConfigSnapshot()

	// Docker hosts with per-entry source
	envDockerNames := ar.manager.EnvDockerHostNames()
	dockerHosts := make([]map[string]any, 0, len(cfg.DockerHosts))
	for _, h := range cfg.DockerHosts {
		source := config.SourceFile
		if envDockerNames[h.Name] {
			source = config.SourceEnv
		}
		dockerHosts = append(dockerHosts, map[string]any{
			"name":   h.Name,
			"host":   h.Host,
			"source": source,
		})
	}

	// Coolify hosts with per-entry source (mask tokens)
	envCoolifyNames := ar.manager.EnvCoolifyHostNames()
	coolifyHosts := make([]map[string]any, 0, len(cfg.CoolifyHosts))
	for _, ch := range cfg.CoolifyHosts {
		source := config.SourceFile
		if envCoolifyNames[ch.HostName] {
			source = config.SourceEnv
		}
		coolifyHosts = append(coolifyHosts, map[string]any{
			"hostName": ch.HostName,
			"apiURL":   ch.APIURL,
			"apiToken": secretMask,
			"source":   source,
		})
	}

	// Auth
	authResp := map[string]any{
		"source":             sources.Auth,
		"enabled":            false,
		"passwordConfigured": false,
	}
	if sources.Auth == config.SourceEnv {
		svc := ar.registry.Auth()
		authResp["enabled"] = svc != nil && !svc.IsDisabled()
		authResp["passwordConfigured"] = svc != nil && !svc.IsDisabled()
	} else if fc.Auth != nil {
		authResp["enabled"] = fc.Auth.Enabled
		authResp["adminUsername"] = fc.Auth.AdminUsername
		authResp["passwordConfigured"] = fc.Auth.AdminPasswordHash != ""
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"dockerHosts": map[string]any{
			"source": sources.DockerHosts,
			"hosts":  dockerHosts,
		},
		"coolifyHosts": map[string]any{
			"source": sources.CoolifyHosts,
			"hosts":  coolifyHosts,
		},
		"readOnly": map[string]any{
			"source": sources.ReadOnly,
			"value":  cfg.ReadOnly,
		},
		"auth": authResp,
	})
}

// UpdateDockerHosts handles PUT /api/v1/settings/docker-hosts.
func (ar *APIRouter) UpdateDockerHosts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Hosts []config.DockerHost `json:"hosts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Hosts) == 0 && len(ar.manager.EnvDockerHostNames()) == 0 {
		http.Error(w, "at least one Docker host is required", http.StatusBadRequest)
		return
	}

	seen := make(map[string]bool)
	for _, h := range req.Hosts {
		if !hostNameRegex.MatchString(h.Name) {
			http.Error(w, fmt.Sprintf("invalid host name: %q", h.Name), http.StatusBadRequest)
			return
		}
		if !isValidHostScheme(h.Host) {
			http.Error(w, fmt.Sprintf("invalid host URL: %q (must start with unix://, ssh://, or tcp://)", h.Host), http.StatusBadRequest)
			return
		}
		if seen[h.Name] {
			http.Error(w, fmt.Sprintf("duplicate host name: %q", h.Name), http.StatusBadRequest)
			return
		}
		seen[h.Name] = true
	}

	if err := ar.manager.UpdateDockerHosts(req.Hosts); err != nil {
		http.Error(w, err.Error(), settingsErrorStatus(err))
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{"message": "Docker hosts updated"})
}

// UpdateCoolifyHosts handles PUT /api/v1/settings/coolify-hosts.
func (ar *APIRouter) UpdateCoolifyHosts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Hosts []struct {
			HostName string `json:"hostName"`
			APIURL   string `json:"apiURL"`
			APIToken string `json:"apiToken"`
		} `json:"hosts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	existing := ar.manager.FileConfigSnapshot()
	existingMap := make(map[string]string)
	for _, ch := range existing.CoolifyHosts {
		existingMap[ch.HostName] = ch.APIToken
	}

	hosts := make([]config.CoolifyHostConfig, 0, len(req.Hosts))
	seen := make(map[string]bool)
	for _, h := range req.Hosts {
		if h.HostName == "" || h.APIURL == "" || h.APIToken == "" {
			http.Error(w, "hostName, apiURL, and apiToken are required for each entry", http.StatusBadRequest)
			return
		}
		if !hostNameRegex.MatchString(h.HostName) {
			http.Error(w, fmt.Sprintf("invalid host name: %q", h.HostName), http.StatusBadRequest)
			return
		}
		if !isValidCoolifyURL(h.APIURL) {
			http.Error(w, fmt.Sprintf("invalid API URL: %q (must start with http:// or https://)", h.APIURL), http.StatusBadRequest)
			return
		}
		if seen[h.HostName] {
			http.Error(w, fmt.Sprintf("duplicate host name: %q", h.HostName), http.StatusBadRequest)
			return
		}
		seen[h.HostName] = true

		token := h.APIToken
		if token == secretMask {
			if stored, ok := existingMap[h.HostName]; ok {
				token = stored
			} else {
				http.Error(w, fmt.Sprintf("no existing token for host %q; provide the actual token", h.HostName), http.StatusBadRequest)
				return
			}
		}

		hosts = append(hosts, config.CoolifyHostConfig{
			HostName: h.HostName,
			APIURL:   strings.TrimRight(h.APIURL, "/"),
			APIToken: token,
		})
	}

	if err := ar.manager.UpdateCoolifyHosts(hosts); err != nil {
		http.Error(w, err.Error(), settingsErrorStatus(err))
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{"message": "Coolify hosts updated"})
}

// UpdateReadOnly handles PUT /api/v1/settings/read-only.
func (ar *APIRouter) UpdateReadOnly(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value bool `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := ar.manager.UpdateReadOnly(req.Value); err != nil {
		http.Error(w, err.Error(), settingsErrorStatus(err))
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{"message": "Read-only mode updated"})
}

// UpdateAuth handles PUT /api/v1/settings/auth.
func (ar *APIRouter) UpdateAuth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled       bool   `json:"enabled"`
		AdminUsername string `json:"adminUsername"`
		NewPassword   string `json:"newPassword,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Enabled && req.AdminUsername == "" {
		http.Error(w, "adminUsername is required when enabling auth", http.StatusBadRequest)
		return
	}

	err := ar.manager.UpdateAuth(func(authCfg *config.FileAuthConfig) (*config.FileAuthConfig, error) {
		authCfg.Enabled = req.Enabled

		if req.Enabled {
			authCfg.AdminUsername = req.AdminUsername

			if authCfg.JWTSecret == "" {
				secret, err := auth.GenerateRandomHex(32)
				if err != nil {
					return nil, fmt.Errorf("failed to generate JWT secret")
				}
				authCfg.JWTSecret = secret
			}

			if req.NewPassword != "" {
				hash, err := auth.HashPassword(req.NewPassword)
				if err != nil {
					return nil, fmt.Errorf("failed to hash password: %w", err)
				}
				authCfg.AdminPasswordHash = hash
				authCfg.AdminPasswordSalt = ""
			}

			if authCfg.AdminPasswordHash == "" {
				return nil, fmt.Errorf("newPassword is required when first enabling auth")
			}
		}

		return authCfg, nil
	})

	if err != nil {
		http.Error(w, err.Error(), settingsErrorStatus(err))
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{"message": "Auth settings updated"})
}

// TestDockerHost handles POST /api/v1/settings/test/docker-host.
func (ar *APIRouter) TestDockerHost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Host string `json:"host"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Host == "" {
		http.Error(w, "host is required", http.StatusBadRequest)
		return
	}

	if !isValidHostScheme(req.Host) {
		http.Error(w, "invalid host URL (must start with unix://, ssh://, or tcp://)", http.StatusBadRequest)
		return
	}

	tempClient, err := docker.NewMultiHostClient([]config.DockerHost{
		{Name: "test", Host: req.Host},
	})
	if err != nil {
		WriteJsonResponse(w, http.StatusOK, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Failed to create client: %v", err),
		})
		return
	}
	defer tempClient.Close()

	cl, err := tempClient.GetClient("test")
	if err != nil {
		WriteJsonResponse(w, http.StatusOK, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Internal error: %v", err),
		})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	ping, err := cl.Ping(ctx)
	if err != nil {
		WriteJsonResponse(w, http.StatusOK, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"success":       true,
		"message":       "Connection successful",
		"dockerVersion": ping.APIVersion,
	})
}

// TestCoolifyHost handles POST /api/v1/settings/test/coolify-host.
func (ar *APIRouter) TestCoolifyHost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		HostName string `json:"hostName"`
		APIURL   string `json:"apiURL"`
		APIToken string `json:"apiToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.HostName == "" || req.APIToken == "" {
		http.Error(w, "hostName and apiToken are required", http.StatusBadRequest)
		return
	}

	cfg := ar.registry.Config()
	var allowedAPIURL string
	for _, hostCfg := range cfg.CoolifyHosts {
		if hostCfg.HostName == req.HostName {
			allowedAPIURL = hostCfg.APIURL
			break
		}
	}
	if allowedAPIURL == "" {
		http.Error(w, "unknown hostName; save the host first before testing", http.StatusBadRequest)
		return
	}

	if req.APIURL != "" && strings.TrimRight(req.APIURL, "/") != strings.TrimRight(allowedAPIURL, "/") {
		http.Error(w, "apiURL does not match configured host URL", http.StatusBadRequest)
		return
	}

	token := req.APIToken
	if token == secretMask {
		fc := ar.manager.FileConfigSnapshot()
		for _, ch := range fc.CoolifyHosts {
			if ch.HostName == req.HostName {
				token = ch.APIToken
				break
			}
		}
		if token == secretMask {
			http.Error(w, "no stored token found; provide the actual token", http.StatusBadRequest)
			return
		}
	}

	client, err := coolify.NewSingleClient(strings.TrimRight(allowedAPIURL, "/"), token)
	if err != nil {
		WriteJsonResponse(w, http.StatusOK, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := client.TestConnection(ctx); err != nil {
		log.Printf("Coolify test connection failed: %v", err)
		WriteJsonResponse(w, http.StatusOK, map[string]any{
			"success": false,
			"message": fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Connection successful",
	})
}

func settingsErrorStatus(err error) int {
	if strings.Contains(err.Error(), "environment variable") {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

func isValidHostScheme(host string) bool {
	return strings.HasPrefix(host, "unix://") ||
		strings.HasPrefix(host, "ssh://") ||
		strings.HasPrefix(host, "tcp://")
}

func isValidCoolifyURL(raw string) bool {
	return strings.HasPrefix(raw, "https://") || strings.HasPrefix(raw, "http://")
}
