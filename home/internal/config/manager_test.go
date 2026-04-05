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

// ─── UpdateScannerConfig ──────────────────────────────────────────────────────

// TestUpdateScannerConfigPersistsAndMerges verifies that UpdateScannerConfig
// writes the file config and reflects the change in the merged config.
func TestUpdateScannerConfigPersistsAndMerges(t *testing.T) {
	m := &Manager{
		envSnapshot: EnvSnapshot{},
		envConfig:   NewConfig(),
		filePath:    filepath.Join(t.TempDir(), "config.json"),
	}
	m.merged, m.sources = m.merge()

	scanner := &FileScannerConfig{
		GrypeImage:     "anchore/grype:v1.0.0",
		DefaultScanner: "trivy",
	}

	if err := m.UpdateScannerConfig(scanner); err != nil {
		t.Fatalf("UpdateScannerConfig returned unexpected error: %v", err)
	}

	merged := m.Config()
	if merged.Scanner.GrypeImage != "anchore/grype:v1.0.0" {
		t.Fatalf("expected GrypeImage 'anchore/grype:v1.0.0', got %q", merged.Scanner.GrypeImage)
	}
	if merged.Scanner.DefaultScanner != "trivy" {
		t.Fatalf("expected DefaultScanner 'trivy', got %q", merged.Scanner.DefaultScanner)
	}
}

// TestUpdateScannerConfigRollsBackOnPersistFailure verifies that when persist
// fails the in-memory state is restored to its previous value.
func TestUpdateScannerConfigRollsBackOnPersistFailure(t *testing.T) {
	// Use an invalid path so persist always fails.
	m := &Manager{
		envSnapshot: EnvSnapshot{},
		envConfig:   NewConfig(),
		filePath:    "/nonexistent/path/config.json",
	}
	m.merged, m.sources = m.merge()

	// Record the original scanner config.
	original := m.fileConfig.Scanner

	err := m.UpdateScannerConfig(&FileScannerConfig{GrypeImage: "shouldNotPersist"})
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}

	// Verify rollback.
	if m.fileConfig.Scanner != original {
		t.Fatal("expected fileConfig.Scanner to be rolled back after persist failure")
	}
}

// TestUpdateScannerConfigWithNotifications verifies that notification settings
// are persisted and reflected in the merged config.
func TestUpdateScannerConfigWithNotifications(t *testing.T) {
	m := &Manager{
		envSnapshot: EnvSnapshot{},
		envConfig:   NewConfig(),
		filePath:    filepath.Join(t.TempDir(), "config.json"),
	}
	m.merged, m.sources = m.merge()

	falseVal := false
	scanner := &FileScannerConfig{
		Notifications: &FileNotificationConfig{
			DiscordWebhookURL: "https://discord.example.com/hook",
			MinSeverity:       "Critical",
			OnScanComplete:    &falseVal,
		},
	}

	if err := m.UpdateScannerConfig(scanner); err != nil {
		t.Fatalf("UpdateScannerConfig returned error: %v", err)
	}

	merged := m.Config()
	if merged.Scanner.DiscordWebhookURL != "https://discord.example.com/hook" {
		t.Fatalf("expected discord webhook to be set, got %q", merged.Scanner.DiscordWebhookURL)
	}
	if merged.Scanner.NotifyMinSeverity != "Critical" {
		t.Fatalf("expected NotifyMinSeverity 'Critical', got %q", merged.Scanner.NotifyMinSeverity)
	}
	if merged.Scanner.NotifyOnComplete {
		t.Fatal("expected NotifyOnComplete=false after setting OnScanComplete=false")
	}
}

// ─── Scanner merge ────────────────────────────────────────────────────────────

// TestScannerMergeUsesEnvConfigAsBase verifies that when no file scanner config
// is present, the merged scanner config equals the env config defaults.
func TestScannerMergeUsesEnvConfigAsBase(t *testing.T) {
	envCfg := NewConfig()
	envCfg.Scanner.GrypeImage = "anchore/grype:env"
	envCfg.Scanner.DefaultScanner = "grype"

	m := &Manager{
		envSnapshot: EnvSnapshot{},
		envConfig:   envCfg,
		filePath:    filepath.Join(t.TempDir(), "config.json"),
	}
	m.merged, m.sources = m.merge()

	merged := m.Config()
	if merged.Scanner.GrypeImage != "anchore/grype:env" {
		t.Fatalf("expected GrypeImage from env, got %q", merged.Scanner.GrypeImage)
	}
	if merged.Scanner.DefaultScanner != "grype" {
		t.Fatalf("expected DefaultScanner from env, got %q", merged.Scanner.DefaultScanner)
	}
}

// TestScannerMergeFileOverridesEnv verifies that non-empty file scanner fields
// override env-derived scanner fields.
func TestScannerMergeFileOverridesEnv(t *testing.T) {
	envCfg := NewConfig()
	envCfg.Scanner.GrypeImage = "anchore/grype:env"
	envCfg.Scanner.TrivyImage = "aquasec/trivy:env"

	m := &Manager{
		envSnapshot: EnvSnapshot{},
		envConfig:   envCfg,
		filePath:    filepath.Join(t.TempDir(), "config.json"),
		fileConfig: FileConfig{
			Scanner: &FileScannerConfig{
				GrypeImage: "anchore/grype:file",
				// TrivyImage deliberately omitted - env value should persist
			},
		},
	}
	m.merged, m.sources = m.merge()

	merged := m.Config()
	if merged.Scanner.GrypeImage != "anchore/grype:file" {
		t.Fatalf("expected file GrypeImage, got %q", merged.Scanner.GrypeImage)
	}
	// TrivyImage not set in file → env value preserved
	if merged.Scanner.TrivyImage != "aquasec/trivy:env" {
		t.Fatalf("expected env TrivyImage preserved, got %q", merged.Scanner.TrivyImage)
	}
}

// TestScannerMergeEmptyFileFieldsDoNotOverrideEnv verifies that empty-string
// file scanner fields do not override non-empty env-derived values.
func TestScannerMergeEmptyFileFieldsDoNotOverrideEnv(t *testing.T) {
	envCfg := NewConfig()
	envCfg.Scanner.DefaultScanner = "trivy"

	m := &Manager{
		envSnapshot: EnvSnapshot{},
		envConfig:   envCfg,
		filePath:    filepath.Join(t.TempDir(), "config.json"),
		fileConfig: FileConfig{
			Scanner: &FileScannerConfig{
				DefaultScanner: "", // empty — must not override
			},
		},
	}
	m.merged, m.sources = m.merge()

	merged := m.Config()
	if merged.Scanner.DefaultScanner != "trivy" {
		t.Fatalf("expected empty file field to keep env value 'trivy', got %q", merged.Scanner.DefaultScanner)
	}
}

// TestScannerMergeNotificationBoolOverride verifies that a pointer-bool
// notification field (OnScanComplete/OnBulkComplete) in the file config
// overrides the env default even when the value is false.
func TestScannerMergeNotificationBoolOverride(t *testing.T) {
	envCfg := NewConfig()
	envCfg.Scanner.NotifyOnComplete = true // env default

	falseVal := false
	m := &Manager{
		envSnapshot: EnvSnapshot{},
		envConfig:   envCfg,
		filePath:    filepath.Join(t.TempDir(), "config.json"),
		fileConfig: FileConfig{
			Scanner: &FileScannerConfig{
				Notifications: &FileNotificationConfig{
					OnScanComplete: &falseVal,
				},
			},
		},
	}
	m.merged, m.sources = m.merge()

	merged := m.Config()
	if merged.Scanner.NotifyOnComplete {
		t.Fatal("expected NotifyOnComplete=false from file config, got true")
	}
}

// ─── parseScannerConfig defaults ─────────────────────────────────────────────

// TestParseScannerConfigDefaults verifies that parseScannerConfig returns the
// expected defaults when no env vars are set.
func TestParseScannerConfigDefaults(t *testing.T) {
	// Ensure scanner env vars are unset for this test.
	vars := []string{
		"SCANNER_GRYPE_IMAGE", "SCANNER_TRIVY_IMAGE", "SCANNER_SYFT_IMAGE",
		"SCANNER_DEFAULT", "SCANNER_GRYPE_ARGS", "SCANNER_TRIVY_ARGS",
		"SCANNER_DISCORD_WEBHOOK_URL", "SCANNER_SLACK_WEBHOOK_URL",
		"SCANNER_NOTIFY_ON_COMPLETE", "SCANNER_NOTIFY_ON_BULK",
		"SCANNER_NOTIFY_MIN_SEVERITY",
	}
	for _, v := range vars {
		t.Setenv(v, "")
	}

	cfg := parseScannerConfig()

	if cfg.GrypeImage == "" {
		t.Fatal("expected default GrypeImage to be non-empty")
	}
	if cfg.TrivyImage == "" {
		t.Fatal("expected default TrivyImage to be non-empty")
	}
	if cfg.SyftImage == "" {
		t.Fatal("expected default SyftImage to be non-empty")
	}
	if cfg.DefaultScanner != "grype" {
		t.Fatalf("expected default scanner 'grype', got %q", cfg.DefaultScanner)
	}
	if !cfg.NotifyOnComplete {
		t.Fatal("expected NotifyOnComplete=true by default")
	}
	if !cfg.NotifyOnBulk {
		t.Fatal("expected NotifyOnBulk=true by default")
	}
	if cfg.NotifyMinSeverity != "High" {
		t.Fatalf("expected default NotifyMinSeverity 'High', got %q", cfg.NotifyMinSeverity)
	}
}

// TestParseScannerConfigEnvOverrides verifies that environment variables
// override the built-in defaults.
func TestParseScannerConfigEnvOverrides(t *testing.T) {
	t.Setenv("SCANNER_GRYPE_IMAGE", "anchore/grype:custom")
	t.Setenv("SCANNER_DEFAULT", "trivy")
	t.Setenv("SCANNER_NOTIFY_ON_COMPLETE", "false")
	t.Setenv("SCANNER_NOTIFY_MIN_SEVERITY", "Critical")

	cfg := parseScannerConfig()

	if cfg.GrypeImage != "anchore/grype:custom" {
		t.Fatalf("expected SCANNER_GRYPE_IMAGE override, got %q", cfg.GrypeImage)
	}
	if cfg.DefaultScanner != "trivy" {
		t.Fatalf("expected SCANNER_DEFAULT override, got %q", cfg.DefaultScanner)
	}
	if cfg.NotifyOnComplete {
		t.Fatal("expected SCANNER_NOTIFY_ON_COMPLETE=false to disable notifications")
	}
	if cfg.NotifyMinSeverity != "Critical" {
		t.Fatalf("expected NotifyMinSeverity 'Critical', got %q", cfg.NotifyMinSeverity)
	}
}