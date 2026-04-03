package config

import (
	"errors"
	"path/filepath"
	"testing"
)

// TestErrEnvironmentConfiguredSentinel ensures the sentinel error is defined and
// behaves correctly with errors.Is for wrapped variants.
func TestErrEnvironmentConfiguredSentinel(t *testing.T) {
	if ErrEnvironmentConfigured == nil {
		t.Fatal("ErrEnvironmentConfigured must not be nil")
	}
	if !errors.Is(ErrEnvironmentConfigured, ErrEnvironmentConfigured) {
		t.Fatal("ErrEnvironmentConfigured must match itself via errors.Is")
	}
}

// TestUpdateReadOnlyReturnsErrEnvironmentConfiguredWhenEnvSet verifies the error
// wrapping change introduced in this PR: UpdateReadOnly must wrap
// ErrEnvironmentConfigured so callers can use errors.Is.
func TestUpdateReadOnlyReturnsErrEnvironmentConfiguredWhenEnvSet(t *testing.T) {
	m := &Manager{
		envSnapshot: EnvSnapshot{ReadOnlySet: true},
		envConfig:   NewConfig(),
		filePath:    filepath.Join(t.TempDir(), "config.json"),
	}
	m.merged, m.sources = m.merge()

	err := m.UpdateReadOnly(true)
	if err == nil {
		t.Fatal("expected error when ReadOnly is env-configured, got nil")
	}
	if !errors.Is(err, ErrEnvironmentConfigured) {
		t.Fatalf("expected errors.Is(err, ErrEnvironmentConfigured)=true, got err=%v", err)
	}
}

// TestUpdateAuthReturnsErrEnvironmentConfiguredWhenEnvSet verifies the same
// wrapping for UpdateAuth.
func TestUpdateAuthReturnsErrEnvironmentConfiguredWhenEnvSet(t *testing.T) {
	m := &Manager{
		envSnapshot: EnvSnapshot{AuthSet: true},
		envConfig:   NewConfig(),
		filePath:    filepath.Join(t.TempDir(), "config.json"),
	}
	m.merged, m.sources = m.merge()

	err := m.UpdateAuth(func(c *FileAuthConfig) (*FileAuthConfig, error) {
		return c, nil
	})
	if err == nil {
		t.Fatal("expected error when Auth is env-configured, got nil")
	}
	if !errors.Is(err, ErrEnvironmentConfigured) {
		t.Fatalf("expected errors.Is(err, ErrEnvironmentConfigured)=true, got err=%v", err)
	}
}

// TestUpdateDockerHostsReturnsErrEnvironmentConfiguredForEnvHost verifies that
// attempting to manage a Docker host that is already defined via env var returns
// an error wrapping ErrEnvironmentConfigured.
func TestUpdateDockerHostsReturnsErrEnvironmentConfiguredForEnvHost(t *testing.T) {
	envCfg := &Config{
		DockerHosts: []DockerHost{
			{Name: "prod", Host: "unix:///var/run/docker.sock"},
		},
	}
	m := &Manager{
		envSnapshot: EnvSnapshot{DockerHostsSet: true},
		envConfig:   envCfg,
		filePath:    filepath.Join(t.TempDir(), "config.json"),
	}
	m.merged, m.sources = m.merge()

	err := m.UpdateDockerHosts([]DockerHost{
		{Name: "prod", Host: "unix:///var/run/docker.sock"},
	})
	if err == nil {
		t.Fatal("expected error when updating an env-defined Docker host, got nil")
	}
	if !errors.Is(err, ErrEnvironmentConfigured) {
		t.Fatalf("expected errors.Is(err, ErrEnvironmentConfigured)=true, got err=%v", err)
	}
}

// TestUpdateDockerHostsAllowsNonEnvHosts verifies that non-conflicting hosts can
// be saved when env hosts are set (the file host has a different name).
func TestUpdateDockerHostsAllowsNonEnvHosts(t *testing.T) {
	envCfg := &Config{
		DockerHosts: []DockerHost{
			{Name: "prod", Host: "unix:///var/run/docker.sock"},
		},
	}
	m := &Manager{
		envSnapshot: EnvSnapshot{DockerHostsSet: true},
		envConfig:   envCfg,
		filePath:    filepath.Join(t.TempDir(), "config.json"),
	}
	m.merged, m.sources = m.merge()

	err := m.UpdateDockerHosts([]DockerHost{
		{Name: "staging", Host: "unix:///var/run/docker.sock"},
	})
	if err != nil {
		t.Fatalf("expected no error for non-conflicting host, got: %v", err)
	}
}

// --- validateCoolifyHosts ---

func TestValidateCoolifyHostsEmptySlice(t *testing.T) {
	if err := validateCoolifyHosts(nil); err != nil {
		t.Fatalf("expected nil error for empty hosts, got: %v", err)
	}
	if err := validateCoolifyHosts([]CoolifyHostConfig{}); err != nil {
		t.Fatalf("expected nil error for empty slice, got: %v", err)
	}
}

func TestValidateCoolifyHostsRejectsEmptyHostName(t *testing.T) {
	err := validateCoolifyHosts([]CoolifyHostConfig{
		{HostName: "", APIURL: "https://coolify.example.com", APIToken: "tok"},
	})
	if err == nil {
		t.Fatal("expected error for empty host name, got nil")
	}
}

func TestValidateCoolifyHostsRejectsWhitespaceOnlyHostName(t *testing.T) {
	err := validateCoolifyHosts([]CoolifyHostConfig{
		{HostName: "   ", APIURL: "https://coolify.example.com", APIToken: "tok"},
	})
	if err == nil {
		t.Fatal("expected error for whitespace-only host name, got nil")
	}
}

func TestValidateCoolifyHostsRejectsDuplicateHostNames(t *testing.T) {
	err := validateCoolifyHosts([]CoolifyHostConfig{
		{HostName: "alpha", APIURL: "https://coolify1.example.com", APIToken: "tok"},
		{HostName: "alpha", APIURL: "https://coolify2.example.com", APIToken: "tok"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate host name, got nil")
	}
}

func TestValidateCoolifyHostsRejectsInvalidScheme(t *testing.T) {
	err := validateCoolifyHosts([]CoolifyHostConfig{
		{HostName: "myhost", APIURL: "ftp://coolify.example.com", APIToken: "tok"},
	})
	if err == nil {
		t.Fatal("expected error for non-https scheme, got nil")
	}
}

func TestValidateCoolifyHostsRejectsEmptyAPIURL(t *testing.T) {
	err := validateCoolifyHosts([]CoolifyHostConfig{
		{HostName: "myhost", APIURL: "", APIToken: "tok"},
	})
	if err == nil {
		t.Fatal("expected error for empty APIURL, got nil")
	}
}