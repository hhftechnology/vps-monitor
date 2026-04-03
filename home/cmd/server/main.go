package main

import (
	"log"
	"net/http"

	"github.com/hhftechnology/vps-monitor/internal/alerts"
	"github.com/hhftechnology/vps-monitor/internal/api"
	"github.com/hhftechnology/vps-monitor/internal/auth"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/coolify"
	"github.com/hhftechnology/vps-monitor/internal/docker"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/scanner"
	"github.com/hhftechnology/vps-monitor/internal/services"
	"github.com/hhftechnology/vps-monitor/internal/system"
)

func main() {
	system.Init()

	manager := config.NewManager()
	cfg := manager.Config()

	multiHostClient, err := docker.NewMultiHostClient(cfg.DockerHosts)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}

	// Auth: env-based first, then file-based fallback.
	authService, err := auth.NewService()
	if err != nil {
		log.Fatalf("Failed to initialize auth service: %v\nPlease ensure ALL auth environment variables are set: JWT_SECRET, ADMIN_USERNAME, and ADMIN_PASSWORD.", err)
	}
	if authService == nil || authService.IsDisabled() {
		fc := manager.FileConfigSnapshot()
		if fc.Auth != nil && fc.Auth.Enabled {
			authService = auth.NewServiceFromFileConfig(fc.Auth)
		}
	}

	if authService == nil || authService.IsDisabled() {
		log.Println("Authentication is DISABLED - no auth environment variables detected")
		log.Println("   To enable authentication, set: JWT_SECRET, ADMIN_USERNAME, ADMIN_PASSWORD")
	} else {
		log.Println("Authentication is ENABLED")
	}

	if cfg.ReadOnly {
		log.Println("READ-ONLY MODE is ENABLED - all mutating operations are disabled")
		log.Println("   To disable read-only mode, set: READONLY_MODE=false or unset the variable")
	} else {
		log.Println("Read-only mode is DISABLED - all operations are allowed")
	}

	// Coolify client
	coolifyClient, err := coolify.NewMultiClient(cfg.CoolifyHosts)
	if err != nil {
		log.Fatalf("Failed to create Coolify client: %v", err)
	}
	if coolifyClient != nil {
		log.Printf("Coolify integration is ENABLED (%d host configs)", len(cfg.CoolifyHosts))
	} else {
		log.Println("Coolify integration is DISABLED")
	}

	// Alert monitor (vps-monitor specific)
	var alertMonitor *alerts.Monitor
	if cfg.Alerts.Enabled {
		alertMonitor = alerts.NewMonitor(multiHostClient, &cfg.Alerts)
		alertMonitor.Start()
		defer alertMonitor.Stop()
		log.Println("Alert monitoring is ENABLED")
		log.Printf("   CPU threshold: %.1f%%, Memory threshold: %.1f%%, Check interval: %s",
			cfg.Alerts.CPUThreshold, cfg.Alerts.MemoryThreshold, cfg.Alerts.CheckInterval)
		if cfg.Alerts.WebhookURL != "" {
			log.Println("   Webhook notifications are ENABLED")
		}
	} else {
		log.Println("Alert monitoring is DISABLED")
		log.Println("   To enable alerts, set: ALERTS_ENABLED=true")
	}

	registry := services.NewRegistry(multiHostClient, coolifyClient, authService, cfg, alertMonitor)

	// Scanner service
	scannerCfg := &models.ScannerConfig{
		GrypeImage:     cfg.Scanner.GrypeImage,
		TrivyImage:     cfg.Scanner.TrivyImage,
		SyftImage:      cfg.Scanner.SyftImage,
		DefaultScanner: models.ScannerType(cfg.Scanner.DefaultScanner),
		GrypeArgs:      cfg.Scanner.GrypeArgs,
		TrivyArgs:      cfg.Scanner.TrivyArgs,
		Notifications: models.NotificationConfig{
			DiscordWebhookURL: cfg.Scanner.DiscordWebhookURL,
			SlackWebhookURL:   cfg.Scanner.SlackWebhookURL,
			OnScanComplete:    cfg.Scanner.NotifyOnComplete,
			OnBulkComplete:    cfg.Scanner.NotifyOnBulk,
			MinSeverity:       models.SeverityLevel(cfg.Scanner.NotifyMinSeverity),
		},
	}
	scannerService := scanner.NewScannerService(registry, scannerCfg)
	log.Printf("Vulnerability scanner ready (default: %s)", cfg.Scanner.DefaultScanner)

	// Hot-reload callback
	manager.OnChange(func(newCfg *config.Config) {
		registry.UpdateConfig(newCfg)

		// Recreate Docker clients
		newDocker, err := docker.NewMultiHostClient(newCfg.DockerHosts)
		if err != nil {
			log.Printf("Warning: failed to recreate Docker clients after config change: %v", err)
		} else {
			registry.SwapDocker(newDocker)
			if alertMonitor != nil {
				alertMonitor.UpdateDockerClient(newDocker)
			}
		}

		// Recreate Coolify clients
		newCoolify, err := coolify.NewMultiClient(newCfg.CoolifyHosts)
		if err != nil {
			log.Printf("Warning: failed to recreate Coolify clients after config change: %v", err)
		} else {
			registry.SwapCoolify(newCoolify)
		}

		// Recreate auth service from file config (env-based auth is immutable)
		fc := manager.FileConfigSnapshot()
		if manager.Sources().Auth == config.SourceFile && fc.Auth != nil {
			registry.SwapAuth(auth.NewServiceFromFileConfig(fc.Auth))
		}

		// Update scanner configuration
		newScannerCfg := &models.ScannerConfig{
			GrypeImage:     newCfg.Scanner.GrypeImage,
			TrivyImage:     newCfg.Scanner.TrivyImage,
			SyftImage:      newCfg.Scanner.SyftImage,
			DefaultScanner: models.ScannerType(newCfg.Scanner.DefaultScanner),
			GrypeArgs:      newCfg.Scanner.GrypeArgs,
			TrivyArgs:      newCfg.Scanner.TrivyArgs,
			Notifications: models.NotificationConfig{
				DiscordWebhookURL: newCfg.Scanner.DiscordWebhookURL,
				SlackWebhookURL:   newCfg.Scanner.SlackWebhookURL,
				OnScanComplete:    newCfg.Scanner.NotifyOnComplete,
				OnBulkComplete:    newCfg.Scanner.NotifyOnBulk,
				MinSeverity:       models.SeverityLevel(newCfg.Scanner.NotifyMinSeverity),
			},
		}
		scannerService.UpdateConfig(newScannerCfg)

		log.Println("Configuration reloaded successfully")
	})

	routerOpts := &api.RouterOptions{
		AlertMonitor:   alertMonitor,
		ScannerService: scannerService,
	}
	apiRouter := api.NewRouter(registry, manager, routerOpts)

	log.Println("Server starting on :6789")
	if err := http.ListenAndServe(":6789", apiRouter); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
