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
}

type Config struct {
	ReadOnly     bool
	Hostname     string // Optional override for displayed hostname
	DockerHosts  []DockerHost
	CoolifyHosts []CoolifyHostConfig
	Alerts       AlertConfig
}

func NewConfig() *Config {
	isReadOnlyMode := os.Getenv("READONLY_MODE") == "true"
	hostname := os.Getenv("HOSTNAME_OVERRIDE") // Custom display hostname
	dockerHosts := parseDockerHosts()
	coolifyHosts := parseCoolifyHostConfigs()
	alertConfig := parseAlertConfig()

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

	return &Config{
		ReadOnly:     isReadOnlyMode,
		Hostname:     hostname,
		DockerHosts:  dockerHosts,
		CoolifyHosts: coolifyHosts,
		Alerts:       alertConfig,
	}
}

func parseAlertConfig() AlertConfig {
	config := AlertConfig{
		Enabled:         os.Getenv("ALERTS_ENABLED") == "true",
		WebhookURL:      os.Getenv("ALERTS_WEBHOOK_URL"),
		CPUThreshold:    80, // Default: 80%
		MemoryThreshold: 90, // Default: 90%
		CheckInterval:   30 * time.Second,
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

	return config
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
