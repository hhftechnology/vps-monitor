package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hhftechnology/vps-monitor/internal/alerts"
	"github.com/hhftechnology/vps-monitor/internal/api"
	"github.com/hhftechnology/vps-monitor/internal/auth"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/docker"
	"github.com/hhftechnology/vps-monitor/internal/system"
)

func main() {
	system.Init()

	cfg := config.NewConfig()
	fmt.Println("Config", cfg)

	multiHostClient, err := docker.NewMultiHostClient(cfg.DockerHosts)
	if err != nil {
		panic(err)
	}

	authService, err := auth.NewService()
	if err != nil {
		log.Fatalf("Failed to initialize auth service: %v\nPlease ensure ALL auth environment variables are set: JWT_SECRET, ADMIN_USERNAME, and ADMIN_PASSWORD.", err)
	}

	if authService == nil {
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

	// Initialize alert monitor if enabled
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

	routerOpts := &api.RouterOptions{
		AlertMonitor: alertMonitor,
	}
	apiRouter := api.NewRouter(multiHostClient, authService, cfg, routerOpts)

	log.Println("Server starting on :6789")
	if err := http.ListenAndServe(":6789", apiRouter); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
