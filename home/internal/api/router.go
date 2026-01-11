package api

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/hhftechnology/vps-monitor/internal/alerts"
	"github.com/hhftechnology/vps-monitor/internal/api/middleware"
	"github.com/hhftechnology/vps-monitor/internal/auth"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/docker"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/static"
)

// Buffer pool for JSON encoding to reduce allocations
var jsonBufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	},
}

type APIRouter struct {
	router        *chi.Mux
	docker        *docker.MultiHostClient
	authService   *auth.Service
	config        *config.Config
	alertMonitor  *alerts.Monitor
	alertHandlers *AlertHandlers
}

// RouterOptions contains optional dependencies for the router
type RouterOptions struct {
	AlertMonitor *alerts.Monitor
}

func NewRouter(docker *docker.MultiHostClient, authService *auth.Service, config *config.Config, opts *RouterOptions) *chi.Mux {
	r := &APIRouter{
		router:      chi.NewRouter(),
		docker:      docker,
		authService: authService,
		config:      config,
	}

	// Set up alert handlers if monitor is provided
	if opts != nil && opts.AlertMonitor != nil {
		r.alertMonitor = opts.AlertMonitor
		r.alertHandlers = NewAlertHandlers(opts.AlertMonitor, &models.AlertConfigResponse{
			Enabled:         config.Alerts.Enabled,
			CPUThreshold:    config.Alerts.CPUThreshold,
			MemoryThreshold: config.Alerts.MemoryThreshold,
			CheckInterval:   config.Alerts.CheckInterval.String(),
			WebhookEnabled:  config.Alerts.WebhookURL != "",
		})
	} else {
		// Create handlers with nil monitor (alerts disabled)
		r.alertHandlers = NewAlertHandlers(nil, &models.AlertConfigResponse{
			Enabled: false,
		})
	}

	return r.Routes()
}

// WriteJsonResponse writes a JSON response using pooled buffers to reduce allocations
func WriteJsonResponse(w http.ResponseWriter, status int, data interface{}) {
	buf := jsonBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer jsonBufferPool.Put(buf)

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(data); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, _ = w.Write(buf.Bytes())
}

func (ar *APIRouter) Routes() *chi.Mux {
	ar.router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}))

	// API routes
	ar.router.Route("/api/v1", func(r chi.Router) {
		// System stats - publicly available for now
		r.Get("/system/stats", ar.GetSystemStats)

		if ar.authService != nil {
			authHandlers := NewAuthHandlers(ar.authService)
			r.Post("/auth/login", authHandlers.Login)

			r.Group(func(protected chi.Router) {
				protected.Use(auth.Middleware(ar.authService))

				protected.Get("/auth/me", authHandlers.GetMe)
				// protected.Get("/system/stats", ar.GetSystemStats) // Moved to public
				ar.registerContainerRoutes(protected)
				ar.registerImageRoutes(protected)
				ar.registerNetworkRoutes(protected)
				ar.registerAlertRoutes(protected)
			})
			return
		}

		// r.Get("/system/stats", ar.GetSystemStats) // Already registered above
		ar.registerContainerRoutes(r)
		ar.registerImageRoutes(r)
		ar.registerNetworkRoutes(r)
		ar.registerAlertRoutes(r)
	})

	// Serve embedded frontend static files
	// This handles all non-API routes and serves the React SPA
	staticFS, err := static.GetFileSystem()
	if err != nil {
		log.Printf("Warning: Could not load embedded frontend files: %v", err)
		log.Println("The frontend will not be available. API routes will still work.")
	} else {
		spaHandler := static.NewSPAHandler(staticFS)
		ar.router.Handle("/*", spaHandler)
	}

	return ar.router
}

func (ar *APIRouter) registerContainerRoutes(r chi.Router) {
	r.Get("/containers", ar.GetContainers)
	r.Route("/containers/{id}", func(r chi.Router) {
		// Read-only routes (always available)
		r.Get("/", ar.GetContainer)
		r.Get("/logs/parsed", ar.GetContainerLogsParsed)
		r.Get("/env", ar.GetEnvVariables)
		r.Get("/stats", ar.HandleContainerStats)       // WebSocket for streaming stats
		r.Get("/stats/once", ar.GetContainerStatsOnce) // Single snapshot

		// Mutating routes (blocked in read-only mode)
		r.Group(func(mutating chi.Router) {
			mutating.Use(middleware.ReadOnly(ar.config))
			mutating.Post("/start", ar.StartContainer)
			mutating.Post("/stop", ar.StopContainer)
			mutating.Post("/restart", ar.RestartContainer)
			mutating.Post("/remove", ar.RemoveContainer)
			mutating.Put("/env", ar.UpdateEnvVariables)
			mutating.Get("/exec", ar.HandleTerminal)
		})
	})
}

func (ar *APIRouter) registerImageRoutes(r chi.Router) {
	r.Get("/images", ar.GetImages)
	r.Route("/images/{id}", func(r chi.Router) {
		r.Get("/", ar.GetImage)

		// Mutating routes (blocked in read-only mode)
		r.Group(func(mutating chi.Router) {
			mutating.Use(middleware.ReadOnly(ar.config))
			mutating.Delete("/", ar.RemoveImage)
		})
	})

	// Image pull (mutating)
	r.Group(func(mutating chi.Router) {
		mutating.Use(middleware.ReadOnly(ar.config))
		mutating.Post("/images/pull", ar.PullImage)
	})
}

func (ar *APIRouter) registerNetworkRoutes(r chi.Router) {
	r.Get("/networks", ar.GetNetworks)
	r.Get("/networks/{id}", ar.GetNetwork)
}

func (ar *APIRouter) registerAlertRoutes(r chi.Router) {
	r.Get("/alerts", ar.alertHandlers.GetAlerts)
	r.Get("/alerts/config", ar.alertHandlers.GetAlertConfig)
	r.Post("/alerts/{id}/acknowledge", ar.alertHandlers.AcknowledgeAlert)
	r.Post("/alerts/acknowledge-all", ar.alertHandlers.AcknowledgeAllAlerts)
}
