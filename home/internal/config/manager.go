package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileConfig represents the JSON config file structure.
type FileConfig struct {
	DockerHosts  []DockerHost        `json:"dockerHosts,omitempty"`
	CoolifyHosts []CoolifyHostConfig `json:"coolifyHosts,omitempty"`
	ReadOnly     *bool               `json:"readOnly,omitempty"`
	Auth         *FileAuthConfig     `json:"auth,omitempty"`
}

// Source indicates where a config value came from.
type Source string

const (
	SourceEnv     Source = "env"
	SourceFile    Source = "file"
	SourceDefault Source = "default"
	SourceMixed   Source = "mixed"
)

var ErrEnvironmentConfigured = errors.New("configured via environment variable")

// EnvSnapshot captures which env vars are set at startup.
type EnvSnapshot struct {
	DockerHostsSet bool
	CoolifySet     bool
	ReadOnlySet    bool
	AuthSet        bool
}

// Manager handles loading, merging, and persisting configuration.
type Manager struct {
	mu          sync.RWMutex
	filePath    string
	envSnapshot EnvSnapshot
	envConfig   *Config         // config derived purely from env vars
	fileConfig  FileConfig      // config from file
	merged      *Config         // final merged config
	sources     ConfigSources   // per-category source tracking
	onChange    []func(*Config) // callbacks when config changes
	generation  uint64          // incremented on each merge to detect stale callbacks
}

// ConfigSources tracks the source of each config category.
type ConfigSources struct {
	DockerHosts  Source `json:"dockerHosts"`
	CoolifyHosts Source `json:"coolifyHosts"`
	ReadOnly     Source `json:"readOnly"`
	Auth         Source `json:"auth"`
}

// NewManager creates a config manager that loads from env vars and an optional file.
func NewManager() *Manager {
	filePath := os.Getenv("CONFIG_PATH")
	if filePath == "" {
		filePath = "/data/config.json"
	}

	envSnapshot := EnvSnapshot{
		DockerHostsSet: os.Getenv("DOCKER_HOSTS") != "",
		CoolifySet:     os.Getenv("COOLIFY_CONFIGS") != "",
		ReadOnlySet:    os.Getenv("READONLY_MODE") != "",
		AuthSet: os.Getenv("JWT_SECRET") != "" ||
			os.Getenv("ADMIN_USERNAME") != "" ||
			os.Getenv("ADMIN_PASSWORD") != "",
	}

	// Load env-based config using existing parsers.
	envConfig := NewConfig()

	m := &Manager{
		filePath:    filePath,
		envSnapshot: envSnapshot,
		envConfig:   envConfig,
	}

	// Load file config (if it exists).
	m.fileConfig = m.loadFile()

	// Merge and compute sources.
	m.merged, m.sources = m.merge()

	return m
}

// Config returns the current merged config (thread-safe).
func (m *Manager) Config() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.merged
}

// Sources returns the source tracking for each category.
func (m *Manager) Sources() ConfigSources {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sources
}

// FileConfigSnapshot returns the current file config (for reading stored secrets).
func (m *Manager) FileConfigSnapshot() FileConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fileConfig
}

// OnChange registers a callback invoked after any config update.
func (m *Manager) OnChange(fn func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = append(m.onChange, fn)
}

// EnvDockerHostNames returns the set of Docker host names defined via env vars.
func (m *Manager) EnvDockerHostNames() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make(map[string]bool)
	if m.envSnapshot.DockerHostsSet {
		for _, h := range m.envConfig.DockerHosts {
			names[h.Name] = true
		}
	}
	return names
}

// EnvCoolifyHostNames returns the set of Coolify host names defined via env vars.
func (m *Manager) EnvCoolifyHostNames() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make(map[string]bool)
	if m.envSnapshot.CoolifySet {
		for _, h := range m.envConfig.CoolifyHosts {
			names[h.HostName] = true
		}
	}
	return names
}

// UpdateDockerHosts updates the file-defined Docker hosts.
func (m *Manager) UpdateDockerHosts(hosts []DockerHost) error {
	m.mu.Lock()

	if m.envSnapshot.DockerHostsSet {
		envNames := make(map[string]bool)
		for _, h := range m.envConfig.DockerHosts {
			envNames[h.Name] = true
		}
		for _, h := range hosts {
			if envNames[h.Name] {
				m.mu.Unlock()
				return fmt.Errorf("%w: host %q is defined via environment variable and cannot be managed from the UI", ErrEnvironmentConfigured, h.Name)
			}
		}
	}

	oldDockerHosts := m.fileConfig.DockerHosts
	m.fileConfig.DockerHosts = hosts
	if err := m.persist(); err != nil {
		m.fileConfig.DockerHosts = oldDockerHosts
		m.mu.Unlock()
		return err
	}
	m.remerge()
	return nil
}

// UpdateCoolifyHosts updates the file-defined Coolify hosts.
func (m *Manager) UpdateCoolifyHosts(hosts []CoolifyHostConfig) error {
	if err := validateCoolifyHosts(hosts); err != nil {
		return err
	}

	m.mu.Lock()

	if m.envSnapshot.CoolifySet {
		envNames := make(map[string]bool)
		for _, h := range m.envConfig.CoolifyHosts {
			envNames[h.HostName] = true
		}
		for _, h := range hosts {
			if envNames[h.HostName] {
				m.mu.Unlock()
				return fmt.Errorf("%w: coolify host %q is defined via environment variable and cannot be managed from the UI", ErrEnvironmentConfigured, h.HostName)
			}
		}
	}

	oldCoolifyHosts := m.fileConfig.CoolifyHosts
	m.fileConfig.CoolifyHosts = hosts
	if err := m.persist(); err != nil {
		m.fileConfig.CoolifyHosts = oldCoolifyHosts
		m.mu.Unlock()
		return err
	}
	m.remerge()
	return nil
}

func validateCoolifyHosts(hosts []CoolifyHostConfig) error {
	seen := make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		hostName := strings.TrimSpace(h.HostName)
		if hostName == "" {
			return fmt.Errorf("coolify host name cannot be empty")
		}
		if _, exists := seen[hostName]; exists {
			return fmt.Errorf("duplicate coolify host name: %q", hostName)
		}
		seen[hostName] = struct{}{}
		if err := validateCoolifyAPIURL(h.APIURL); err != nil {
			return fmt.Errorf("invalid API URL for host %q: %w", hostName, err)
		}
	}
	return nil
}

func validateCoolifyAPIURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}

	allowInsecureHTTP := os.Getenv("COOLIFY_ALLOW_INSECURE_HTTP") == "true"
	if parsed.Scheme != "https" && !(allowInsecureHTTP && parsed.Scheme == "http") {
		return fmt.Errorf("invalid API URL scheme: only https is allowed by default (set COOLIFY_ALLOW_INSECURE_HTTP=true to allow http)")
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

// UpdateReadOnly updates the read-only setting in the file config.
func (m *Manager) UpdateReadOnly(readOnly bool) error {
	m.mu.Lock()

	if m.envSnapshot.ReadOnlySet {
		m.mu.Unlock()
		return fmt.Errorf("%w: read-only mode is configured via environment variable and cannot be changed from the UI", ErrEnvironmentConfigured)
	}

	oldReadOnly := m.fileConfig.ReadOnly
	m.fileConfig.ReadOnly = &readOnly
	if err := m.persist(); err != nil {
		m.fileConfig.ReadOnly = oldReadOnly
		m.mu.Unlock()
		return err
	}
	m.remerge()
	return nil
}

// UpdateAuth applies a mutation function to the current file auth config atomically.
func (m *Manager) UpdateAuth(mutate func(current *FileAuthConfig) (*FileAuthConfig, error)) error {
	m.mu.Lock()

	if m.envSnapshot.AuthSet {
		m.mu.Unlock()
		return fmt.Errorf("%w: auth is configured via environment variables and cannot be changed from the UI", ErrEnvironmentConfigured)
	}

	oldAuth := m.fileConfig.Auth
	current := &FileAuthConfig{}
	if oldAuth != nil {
		*current = *oldAuth
	}

	updated, err := mutate(current)
	if err != nil {
		m.mu.Unlock()
		return err
	}

	m.fileConfig.Auth = updated
	if err := m.persist(); err != nil {
		m.fileConfig.Auth = oldAuth
		m.mu.Unlock()
		return err
	}
	m.remerge()
	return nil
}

// merge produces the merged config and source tracking. Must be called with lock held.
func (m *Manager) merge() (*Config, ConfigSources) {
	cfg := &Config{}
	sources := ConfigSources{}

	// Preserve vps-monitor specific fields from env config
	cfg.Hostname = m.envConfig.Hostname
	cfg.Alerts = m.envConfig.Alerts

	// Docker hosts: env hosts + file hosts combined. Env hosts win on name collision.
	envDockerNames := make(map[string]bool)
	if m.envSnapshot.DockerHostsSet {
		for _, h := range m.envConfig.DockerHosts {
			cfg.DockerHosts = append(cfg.DockerHosts, h)
			envDockerNames[h.Name] = true
		}
	}
	for _, h := range m.fileConfig.DockerHosts {
		if !envDockerNames[h.Name] {
			cfg.DockerHosts = append(cfg.DockerHosts, h)
		}
	}
	if len(cfg.DockerHosts) == 0 {
		cfg.DockerHosts = []DockerHost{{Name: "local", Host: "unix:///var/run/docker.sock"}}
		sources.DockerHosts = SourceDefault
	} else if m.envSnapshot.DockerHostsSet && len(m.fileConfig.DockerHosts) > 0 {
		sources.DockerHosts = SourceMixed
	} else if m.envSnapshot.DockerHostsSet {
		sources.DockerHosts = SourceEnv
	} else {
		sources.DockerHosts = SourceFile
	}

	// Coolify hosts: same merge strategy.
	envCoolifyNames := make(map[string]bool)
	if m.envSnapshot.CoolifySet {
		for _, h := range m.envConfig.CoolifyHosts {
			cfg.CoolifyHosts = append(cfg.CoolifyHosts, h)
			envCoolifyNames[h.HostName] = true
		}
	}
	for _, h := range m.fileConfig.CoolifyHosts {
		if !envCoolifyNames[h.HostName] {
			cfg.CoolifyHosts = append(cfg.CoolifyHosts, h)
		}
	}
	if len(cfg.CoolifyHosts) == 0 {
		sources.CoolifyHosts = SourceDefault
	} else if m.envSnapshot.CoolifySet && len(m.fileConfig.CoolifyHosts) > 0 {
		sources.CoolifyHosts = SourceMixed
	} else if m.envSnapshot.CoolifySet {
		sources.CoolifyHosts = SourceEnv
	} else {
		sources.CoolifyHosts = SourceFile
	}

	// Read-only
	if m.envSnapshot.ReadOnlySet {
		cfg.ReadOnly = m.envConfig.ReadOnly
		sources.ReadOnly = SourceEnv
	} else if m.fileConfig.ReadOnly != nil {
		cfg.ReadOnly = *m.fileConfig.ReadOnly
		sources.ReadOnly = SourceFile
	} else {
		cfg.ReadOnly = false
		sources.ReadOnly = SourceDefault
	}

	// Auth
	if m.envSnapshot.AuthSet {
		sources.Auth = SourceEnv
	} else if m.fileConfig.Auth != nil {
		sources.Auth = SourceFile
	} else {
		sources.Auth = SourceDefault
	}

	return cfg, sources
}

// remerge re-merges config then unlocks the mutex before firing callbacks.
// Callers must hold the write lock. The lock is released by this function.
func (m *Manager) remerge() {
	m.generation++
	gen := m.generation
	m.merged, m.sources = m.merge()
	cfg := m.merged
	cbs := make([]func(*Config), len(m.onChange))
	copy(cbs, m.onChange)
	m.mu.Unlock()

	m.mu.RLock()
	if m.generation != gen {
		m.mu.RUnlock()
		return
	}
	for _, fn := range cbs {
		fn(cfg)
	}
	m.mu.RUnlock()
}

// loadFile reads the config file from disk.
func (m *Manager) loadFile() FileConfig {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileConfig{}
		}
		log.Fatalf("Failed to read config file %s: %v", m.filePath, err)
	}

	var fc FileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		log.Fatalf("Failed to parse config file %s: %v\nIf the file is corrupted, delete it and restart.", m.filePath, err)
	}
	return fc
}

// persist writes the file config to disk atomically. Must be called with lock held.
func (m *Manager) persist() error {
	data, err := json.MarshalIndent(m.fileConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	tmp := m.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to write temp config file: %w", err)
	}
	if err := os.Rename(tmp, m.filePath); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("failed to rename config file: %w", err)
	}
	return nil
}
