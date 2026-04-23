package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type DockerHost struct {
	Name string `json:"name"`
	Host string `json:"host"`
}

// CoolifyHostConfig maps a Docker host name to a Coolify API instance.
type CoolifyHostConfig struct {
	HostName string `json:"hostName"`
	APIURL   string `json:"apiURL"`
	APIToken string `json:"apiToken"`
}

// FileAuthConfig represents auth settings stored in the config file.
type FileAuthConfig struct {
	Enabled           bool   `json:"enabled"`
	JWTSecret         string `json:"jwtSecret,omitempty"`
	AdminUsername     string `json:"adminUsername,omitempty"`
	AdminPasswordHash string `json:"adminPasswordHash,omitempty"`
	// Deprecated: retained for backward compatibility with legacy SHA-256 password hashes.
	AdminPasswordSalt string `json:"adminPasswordSalt,omitempty"`
}

// AlertConfig holds configuration for the alerting system
type AlertConfig struct {
	Enabled         bool
	WebhookURL      string
	CPUThreshold    float64       // 0-100, alert when exceeded
	MemoryThreshold float64       // 0-100, alert when exceeded
	CheckInterval   time.Duration // How often to check thresholds
	AlertsFilter    string
}

type BotConfig struct {
	Enabled       bool
	Mode          string
	TelegramToken string
	AllowedChatID string
	PollInterval  time.Duration
	Discord       DiscordBotConfig
}

type DiscordBotConfig struct {
	Enabled          bool
	BotToken         string
	ApplicationID    string
	GuildID          string
	AllowedChannelID string
}

const (
	BotModePolling  = "polling"
	BotModeJWTRelay = "jwt-relay"
)

// ScannerConfig holds configuration for vulnerability scanning
type ScannerConfig struct {
	GrypeImage           string
	TrivyImage           string
	SyftImage            string
	DefaultScanner       string
	GrypeArgs            string
	TrivyArgs            string
	DiscordWebhookURL    string
	SlackWebhookURL      string
	NotifyOnComplete     bool
	NotifyOnBulk         bool
	NotifyOnNewCVEs      bool
	NotifyMinSeverity    string
	AutoScanEnabled      bool
	AutoScanPollInterval int // minutes, default 15
	ForceRescan          bool
	ScanTimeoutMinutes   int // default 20
	BulkTimeoutMinutes   int // default 120
	ScannerMemoryMB      int // default 2048
	ScannerPidsLimit     int // default 512
}

type Config struct {
	ReadOnly     bool
	Hostname     string // Optional override for displayed hostname
	DockerHosts  []DockerHost
	CoolifyHosts []CoolifyHostConfig
	Alerts       AlertConfig
	Bot          BotConfig
	Scanner      ScannerConfig
}

func NewConfig() *Config {
	isReadOnlyMode := os.Getenv("READONLY_MODE") == "true"
	hostname := os.Getenv("HOSTNAME_OVERRIDE") // Custom display hostname
	dockerHosts := parseDockerHosts()
	coolifyHosts := parseCoolifyHostConfigs()
	alertConfig := parseAlertConfig()
	botConfig := parseBotConfig()

	// if we don't have any docker hosts, we should default back to
	// the unix socket on the machine running vps-monitor.
	if len(dockerHosts) == 0 {
		dockerHosts = []DockerHost{{Name: "local", Host: "unix:///var/run/docker.sock"}}
	}

	// Warn if Coolify hosts reference unknown Docker hosts
	dockerHostNames := make(map[string]bool, len(dockerHosts))
	for _, dh := range dockerHosts {
		dockerHostNames[dh.Name] = true
	}
	for _, ch := range coolifyHosts {
		if !dockerHostNames[ch.HostName] {
			log.Printf("Warning: COOLIFY_CONFIGS references unknown Docker host %q", ch.HostName)
		}
	}

	scannerConfig := parseScannerConfig()

	return &Config{
		ReadOnly:     isReadOnlyMode,
		Hostname:     hostname,
		DockerHosts:  dockerHosts,
		CoolifyHosts: coolifyHosts,
		Alerts:       alertConfig,
		Bot:          botConfig,
		Scanner:      scannerConfig,
	}
}

func parseAlertConfig() AlertConfig {
	config := AlertConfig{
		Enabled:         os.Getenv("ALERTS_ENABLED") == "true",
		WebhookURL:      os.Getenv("ALERTS_WEBHOOK_URL"),
		CPUThreshold:    80, // Default: 80%
		MemoryThreshold: 90, // Default: 90%
		CheckInterval:   30 * time.Second,
		AlertsFilter:    "all",
	}

	if cpuStr := os.Getenv("ALERTS_CPU_THRESHOLD"); cpuStr != "" {
		if cpu, err := strconv.ParseFloat(cpuStr, 64); err == nil && cpu > 0 && cpu <= 100 {
			config.CPUThreshold = cpu
		}
	}

	if memStr := os.Getenv("ALERTS_MEMORY_THRESHOLD"); memStr != "" {
		if mem, err := strconv.ParseFloat(memStr, 64); err == nil && mem > 0 && mem <= 100 {
			config.MemoryThreshold = mem
		}
	}

	if intervalStr := os.Getenv("ALERTS_CHECK_INTERVAL"); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil && interval > 0 {
			config.CheckInterval = interval
		}
	}

	if filter := strings.TrimSpace(os.Getenv("ALERTS_FILTER")); filter != "" {
		config.AlertsFilter = filter
	}

	return config
}

func parseBotConfig() BotConfig {
	cfg := BotConfig{
		Enabled:       os.Getenv("BOT_ENABLED") == "true",
		Mode:          NormalizeBotMode(os.Getenv("BOT_MODE")),
		TelegramToken: strings.TrimSpace(os.Getenv("BOT_TELEGRAM_TOKEN")),
		AllowedChatID: strings.TrimSpace(os.Getenv("BOT_ALLOWED_CHAT_ID")),
		PollInterval:  15 * time.Second,
		Discord: DiscordBotConfig{
			Enabled:          os.Getenv("BOT_DISCORD_ENABLED") == "true",
			BotToken:         strings.TrimSpace(os.Getenv("BOT_DISCORD_TOKEN")),
			ApplicationID:    strings.TrimSpace(os.Getenv("BOT_DISCORD_APPLICATION_ID")),
			GuildID:          strings.TrimSpace(os.Getenv("BOT_DISCORD_GUILD_ID")),
			AllowedChannelID: strings.TrimSpace(os.Getenv("BOT_DISCORD_ALLOWED_CHANNEL_ID")),
		},
	}

	if intervalStr := strings.TrimSpace(os.Getenv("BOT_POLL_INTERVAL")); intervalStr != "" {
		if interval, err := time.ParseDuration(intervalStr); err == nil && interval > 0 {
			cfg.PollInterval = interval
		}
	}

	if cfg.TelegramToken == "" || cfg.AllowedChatID == "" {
		cfg.Enabled = false
	}
	if cfg.Discord.BotToken == "" || cfg.Discord.ApplicationID == "" || cfg.Discord.AllowedChannelID == "" {
		cfg.Discord.Enabled = false
	}

	return cfg
}

func NormalizeBotMode(raw string) string {
	switch strings.TrimSpace(raw) {
	case "", BotModePolling:
		return BotModePolling
	case BotModeJWTRelay:
		return BotModeJWTRelay
	default:
		return BotModePolling
	}
}

func parseCoolifyHostConfigs() []CoolifyHostConfig {
	// Format: COOLIFY_CONFIGS=hostA|https://coolify-a.com|tokenA,hostB|https://coolify-b.com|tokenB
	raw := os.Getenv("COOLIFY_CONFIGS")
	if raw == "" {
		return nil
	}

	var configs []CoolifyHostConfig
	seen := make(map[string]bool)

	for entry := range strings.SplitSeq(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "|", 3)
		if len(parts) != 3 {
			log.Fatalf("Invalid COOLIFY_CONFIGS format: %s (expected: hostName|apiURL|apiToken)", entry)
		}
		hostName := strings.TrimSpace(parts[0])
		apiURL := strings.TrimRight(strings.TrimSpace(parts[1]), "/")
		apiToken := strings.TrimSpace(parts[2])

		if hostName == "" || apiURL == "" || apiToken == "" {
			log.Fatalf("Invalid COOLIFY_CONFIGS format: %s (all fields must be non-empty)", entry)
		}
		if seen[hostName] {
			log.Fatalf("Duplicate Coolify host name in COOLIFY_CONFIGS: %s", hostName)
		}
		seen[hostName] = true

		configs = append(configs, CoolifyHostConfig{
			HostName: hostName,
			APIURL:   apiURL,
			APIToken: apiToken,
		})
	}

	return configs
}

func parseScannerConfig() ScannerConfig {
	cfg := ScannerConfig{
		GrypeImage:           "anchore/grype:v0.110.0",
		TrivyImage:           "aquasec/trivy:0.69.3",
		SyftImage:            "anchore/syft:v1.42.3",
		DefaultScanner:       "grype",
		GrypeArgs:            "",
		TrivyArgs:            "",
		DiscordWebhookURL:    os.Getenv("SCANNER_DISCORD_WEBHOOK_URL"),
		SlackWebhookURL:      os.Getenv("SCANNER_SLACK_WEBHOOK_URL"),
		NotifyOnComplete:     true,
		NotifyOnBulk:         true,
		NotifyOnNewCVEs:      true,
		NotifyMinSeverity:    "High",
		AutoScanEnabled:      false,
		AutoScanPollInterval: 15,
		ForceRescan:          false,
		ScanTimeoutMinutes:   20,
		BulkTimeoutMinutes:   120,
		ScannerMemoryMB:      2048,
		ScannerPidsLimit:     512,
	}

	if v := os.Getenv("SCANNER_GRYPE_IMAGE"); v != "" {
		cfg.GrypeImage = v
	}
	if v := os.Getenv("SCANNER_TRIVY_IMAGE"); v != "" {
		cfg.TrivyImage = v
	}
	if v := os.Getenv("SCANNER_SYFT_IMAGE"); v != "" {
		cfg.SyftImage = v
	}
	if v := os.Getenv("SCANNER_DEFAULT"); v != "" {
		cfg.DefaultScanner = v
	}
	if v := os.Getenv("SCANNER_GRYPE_ARGS"); v != "" {
		cfg.GrypeArgs = v
	}
	if v := os.Getenv("SCANNER_TRIVY_ARGS"); v != "" {
		cfg.TrivyArgs = v
	}
	if v := os.Getenv("SCANNER_NOTIFY_ON_COMPLETE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.NotifyOnComplete = b
		}
	}
	if v := os.Getenv("SCANNER_NOTIFY_ON_BULK"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.NotifyOnBulk = b
		}
	}
	if v := os.Getenv("SCANNER_NOTIFY_ON_NEW_CVES"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.NotifyOnNewCVEs = b
		}
	}
	if v := os.Getenv("SCANNER_AUTO_SCAN"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.AutoScanEnabled = b
		}
	}
	if v := os.Getenv("SCANNER_FORCE_RESCAN"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.ForceRescan = b
		}
	}
	if v := os.Getenv("SCANNER_NOTIFY_MIN_SEVERITY"); v != "" {
		cfg.NotifyMinSeverity = v
	}
	if v := os.Getenv("SCANNER_AUTO_SCAN_POLL_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.AutoScanPollInterval = n
		}
	}
	if v := os.Getenv("SCANNER_TIMEOUT_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ScanTimeoutMinutes = n
		}
	}
	if v := os.Getenv("SCANNER_BULK_TIMEOUT_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.BulkTimeoutMinutes = n
		}
	}
	if v := os.Getenv("SCANNER_MEMORY_MB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ScannerMemoryMB = n
		}
	}
	if v := os.Getenv("SCANNER_PIDS_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ScannerPidsLimit = n
		}
	}

	if cfg.DefaultScanner != "grype" && cfg.DefaultScanner != "trivy" {
		cfg.DefaultScanner = "grype"
	}
	if cfg.NotifyMinSeverity != "Low" && cfg.NotifyMinSeverity != "Medium" &&
		cfg.NotifyMinSeverity != "High" && cfg.NotifyMinSeverity != "Critical" {
		cfg.NotifyMinSeverity = "High"
	}

	return cfg
}

func parseDockerHosts() []DockerHost {
	// Format: DOCKER_HOSTS=local=unix:///var/run/docker.sock,remote=ssh://root@X.X.X.X
	dockerHosts := os.Getenv("DOCKER_HOSTS")
	if dockerHosts == "" {
		return []DockerHost{}
	}

	dockerHostsList := []DockerHost{}

	dockerHostStrings := strings.SplitSeq(dockerHosts, ",")
	for dockerHostString := range dockerHostStrings {
		parts := strings.SplitN(strings.TrimSpace(dockerHostString), "=", 2)
		if len(parts) != 2 {
			log.Fatalf("Invalid DOCKER_HOSTS format: %s (expected format: name=host)", dockerHostString)
		}

		name := strings.TrimSpace(parts[0])
		host := strings.TrimSpace(parts[1])
		if name == "" || host == "" {
			log.Fatalf("Invalid DOCKER_HOSTS format: %s (name and host cannot be empty)", dockerHostString)
		}

		dockerHostsList = append(dockerHostsList, DockerHost{Name: name, Host: host})
	}

	return dockerHostsList
}
