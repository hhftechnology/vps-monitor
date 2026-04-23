package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/alerts"
	"github.com/hhftechnology/vps-monitor/internal/api"
	"github.com/hhftechnology/vps-monitor/internal/auth"
	"github.com/hhftechnology/vps-monitor/internal/bot"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/containerstats"
	"github.com/hhftechnology/vps-monitor/internal/coolify"
	"github.com/hhftechnology/vps-monitor/internal/docker"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/scanner"
	"github.com/hhftechnology/vps-monitor/internal/services"
	"github.com/hhftechnology/vps-monitor/internal/system"
)

func main() {
	system.Init()

	const containerStatsRetention = 30 * 24 * time.Hour

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

	// Scanner database
	dbPath := "/data/scanner.db"
	if v := os.Getenv("SCANNER_DB_PATH"); v != "" {
		dbPath = v
	}
	scanDB, err := scanner.NewScanDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to open scan database: %v", err)
	}
	defer scanDB.Close()
	log.Printf("Scan database opened at %s", dbPath)

	// Alert monitor / stats collection
	// alertMonitor starts nil and is injected after creation when alerts are enabled.
	var alertMonitor *alerts.Monitor
	registry := services.NewRegistry(multiHostClient, coolifyClient, authService, cfg, alertMonitor)

	var statsCollector *containerstats.Collector
	if cfg.Alerts.Enabled {
		alertMonitor = alerts.NewMonitor(multiHostClient, &cfg.Alerts, scanDB, containerStatsRetention)
		registry.SwapAlerts(alertMonitor)
		alertMonitor.Start()
		defer alertMonitor.Stop()
		log.Println("Alert monitoring is ENABLED")
		log.Printf("   CPU threshold: %.1f%%, Memory threshold: %.1f%%, Check interval: %s",
			cfg.Alerts.CPUThreshold, cfg.Alerts.MemoryThreshold, cfg.Alerts.CheckInterval)
		if cfg.Alerts.WebhookURL != "" {
			log.Println("   Webhook notifications are ENABLED")
		}
	} else {
		statsCollector = containerstats.NewCollector(registry, scanDB, cfg.Stats.SampleInterval, containerStatsRetention)
		statsCollector.Start()
		defer statsCollector.Stop()
		log.Println("Alert monitoring is DISABLED")
		log.Println("   Background container stats collection remains ENABLED")
		log.Println("   To enable alerts, set: ALERTS_ENABLED=true")
	}

	telegramBot := bot.NewService(registry, cfg.Bot)
	telegramBot.Start()
	defer telegramBot.Stop()

	// Build initial scanner config from env, then load/merge with DB settings
	envScannerCfg := configToScannerConfig(cfg.Scanner)
	if err := scanDB.MigrateFromFileConfig(envScannerCfg); err != nil {
		log.Printf("Warning: failed to migrate scanner config to DB: %v", err)
	}
	scannerCfg := scanDB.LoadScannerSettings(envScannerCfg)

	// Scanner service
	scannerService := scanner.NewScannerService(registry, scannerCfg, scanDB)
	log.Printf("Vulnerability scanner ready (default: %s)", scannerCfg.DefaultScanner)

	// Auto-scanner
	autoScanner := scanner.NewAutoScanner(registry, scannerService, scanDB)
	if scannerCfg.AutoScan.Enabled {
		autoScanner.Start()
		log.Println("Auto-scan is ENABLED")
	} else {
		log.Println("Auto-scan is DISABLED")
	}
	defer autoScanner.Stop()

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

		// Update scanner configuration from DB (with env overrides)
		newEnvCfg := configToScannerConfig(newCfg.Scanner)
		newScannerCfg := scanDB.LoadScannerSettings(newEnvCfg)
		scannerService.UpdateConfig(newScannerCfg)

		// Toggle auto-scanner
		if newScannerCfg.AutoScan.Enabled {
			autoScanner.SetPollInterval(newScannerCfg.AutoScan.PollInterval)
			if !autoScanner.IsEnabled() {
				autoScanner.Stop()
				autoScanner.Start()
			}
		} else {
			autoScanner.Stop()
		}

		telegramBot.UpdateConfig(newCfg.Bot)

		log.Println("Configuration reloaded successfully")
	})

	routerOpts := &api.RouterOptions{
		AlertMonitor:   alertMonitor,
		BotService:     telegramBot,
		ScanDB:         scanDB,
		ScannerService: scannerService,
		AutoScanner:    autoScanner,
	}
	apiRouter := api.NewRouter(registry, manager, routerOpts)

	log.Println("Server starting on :6789")
	if err := http.ListenAndServe(":6789", apiRouter); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func configToScannerConfig(sc config.ScannerConfig) *models.ScannerConfig {
	return &models.ScannerConfig{
		GrypeImage:     sc.GrypeImage,
		TrivyImage:     sc.TrivyImage,
		SyftImage:      sc.SyftImage,
		DefaultScanner: models.ScannerType(sc.DefaultScanner),
		GrypeArgs:      sc.GrypeArgs,
		TrivyArgs:      sc.TrivyArgs,
		Notifications: models.NotificationConfig{
			DiscordWebhookURL: sc.DiscordWebhookURL,
			SlackWebhookURL:   sc.SlackWebhookURL,
			OnScanComplete:    sc.NotifyOnComplete,
			OnBulkComplete:    sc.NotifyOnBulk,
			OnNewCVEs:         sc.NotifyOnNewCVEs,
			MinSeverity:       models.SeverityLevel(sc.NotifyMinSeverity),
		},
		AutoScan: models.AutoScanConfig{
			Enabled:      sc.AutoScanEnabled,
			PollInterval: sc.AutoScanPollInterval,
		},
		ForceRescan:        sc.ForceRescan,
		ScanTimeoutMinutes: sc.ScanTimeoutMinutes,
		BulkTimeoutMinutes: sc.BulkTimeoutMinutes,
		ScannerMemoryMB:    sc.ScannerMemoryMB,
		ScannerPidsLimit:   sc.ScannerPidsLimit,
	}
}
