package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type DockerHost struct {
	Name string
	Host string
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
	ReadOnly    bool
	Hostname    string // Optional override for displayed hostname
	DockerHosts []DockerHost
	Alerts      AlertConfig
}

func NewConfig() *Config {
	isReadOnlyMode := os.Getenv("READONLY_MODE") == "true"
	hostname := os.Getenv("HOSTNAME_OVERRIDE") // Custom display hostname
	dockerHosts := parseDockerHosts()
	alertConfig := parseAlertConfig()

	// if we don't have any docker hosts, we should default back to
	// the unix socket on the machine running vps-monitor.
	if len(dockerHosts) == 0 {
		dockerHosts = []DockerHost{{Name: "local", Host: "unix:///var/run/docker.sock"}}
	}

	return &Config{
		ReadOnly:    isReadOnlyMode,
		Hostname:    hostname,
		DockerHosts: dockerHosts,
		Alerts:      alertConfig,
	}
}

func parseAlertConfig() AlertConfig {
	config := AlertConfig{
		Enabled:         os.Getenv("ALERTS_ENABLED") == "true",
		WebhookURL:      os.Getenv("ALERTS_WEBHOOK_URL"),
		CPUThreshold:    80,  // Default: 80%
		MemoryThreshold: 90,  // Default: 90%
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
