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
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/scanner"
	"github.com/hhftechnology/vps-monitor/internal/services"
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
	registry      *services.Registry
	manager       *config.Manager
	alertHandlers *AlertHandlers
	scanHandlers  *ScanHandlers
}

// RouterOptions contains optional dependencies for the router
type RouterOptions struct {
	AlertMonitor   *alerts.Monitor
	ScannerService *scanner.ScannerService
}

func NewRouter(registry *services.Registry, manager *config.Manager, opts *RouterOptions) *chi.Mux {
	cfg := registry.Config()

	r := &APIRouter{
		router:   chi.NewRouter(),
		registry: registry,
		manager:  manager,
	}

	// Set up scan handlers
	if opts != nil && opts.ScannerService != nil {
		r.scanHandlers = NewScanHandlers(opts.ScannerService, manager)
	} else {
		r.scanHandlers = nil
	}

	// Set up alert handlers
	if opts != nil && opts.AlertMonitor != nil {
		r.alertHandlers = NewAlertHandlers(opts.AlertMonitor, &models.AlertConfigResponse{
			Enabled:         cfg.Alerts.Enabled,
			CPUThreshold:    cfg.Alerts.CPUThreshold,
			MemoryThreshold: cfg.Alerts.MemoryThreshold,
			CheckInterval:   cfg.Alerts.CheckInterval.String(),
			WebhookEnabled:  cfg.Alerts.WebhookURL != "",
		})
	} else {
		r.alertHandlers = NewAlertHandlers(nil, &models.AlertConfigResponse{
			Enabled:         false,
			CPUThreshold:    cfg.Alerts.CPUThreshold,
			MemoryThreshold: cfg.Alerts.MemoryThreshold,
			CheckInterval:   cfg.Alerts.CheckInterval.String(),
			WebhookEnabled:  cfg.Alerts.WebhookURL != "",
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
		// System stats - publicly available
		r.Get("/system/stats", ar.GetSystemStats)

		// Auth login - always registered, dynamic behavior
		r.Post("/auth/login", ar.handleLogin)

		// Settings endpoints (protected by dynamic auth)
		ar.registerSettingsRoutes(r)

		// All other routes go through dynamic auth middleware
		r.Group(func(protected chi.Router) {
			protected.Use(auth.DynamicMiddleware(ar.registry.Auth))

			protected.Get("/auth/me", ar.handleGetMe)
			protected.Post("/devices/register", ar.RegisterDevice)
			ar.registerContainerRoutes(protected)
			ar.registerImageRoutes(protected)
			ar.registerNetworkRoutes(protected)
			ar.registerAlertRoutes(protected)
			ar.registerScanRoutes(protected)
		})
	})

	// Serve embedded frontend static files
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
		r.Get("/stats", ar.HandleContainerStats)
		r.Get("/stats/once", ar.GetContainerStatsOnce)

		// Mutating routes (blocked in read-only mode)
		r.Group(func(mutating chi.Router) {
			mutating.Use(middleware.ReadOnly(func() bool {
				return ar.registry.Config().ReadOnly
			}))
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
			mutating.Use(middleware.ReadOnly(func() bool {
				return ar.registry.Config().ReadOnly
			}))
			mutating.Delete("/", ar.RemoveImage)
		})
	})

	// Image pull (mutating)
	r.Group(func(mutating chi.Router) {
		mutating.Use(middleware.ReadOnly(func() bool {
			return ar.registry.Config().ReadOnly
		}))
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

func (ar *APIRouter) registerScanRoutes(r chi.Router) {
	if ar.scanHandlers == nil {
		return
	}

	// Read-only routes
	r.Get("/scan/jobs", ar.scanHandlers.GetScanJobs)
	r.Get("/scan/jobs/{id}", ar.scanHandlers.GetScanJob)
	r.Get("/scan/results", ar.scanHandlers.GetScanResults)
	r.Get("/scan/results/latest", ar.scanHandlers.GetLatestScanResult)
	r.Get("/scan/sbom/{id}", ar.scanHandlers.GetSBOMJob)

	// Mutating routes (blocked in read-only mode)
	r.Group(func(mutating chi.Router) {
		mutating.Use(middleware.ReadOnly(func() bool {
			return ar.registry.Config().ReadOnly
		}))
		mutating.Post("/scan", ar.scanHandlers.StartScan)
		mutating.Post("/scan/bulk", ar.scanHandlers.StartBulkScan)
		mutating.Delete("/scan/jobs/{id}", ar.scanHandlers.CancelScanJob)
		mutating.Post("/scan/sbom", ar.scanHandlers.StartSBOMGeneration)
	})
}

func (ar *APIRouter) registerSettingsRoutes(r chi.Router) {
	r.Route("/settings", func(r chi.Router) {
		r.Use(auth.DynamicMiddleware(ar.registry.Auth))

		r.Get("/", ar.GetSettings)
		r.Put("/docker-hosts", ar.UpdateDockerHosts)
		r.Put("/coolify-hosts", ar.UpdateCoolifyHosts)
		r.Put("/read-only", ar.UpdateReadOnly)
		r.Put("/auth", ar.UpdateAuth)
		r.Post("/test/docker-host", ar.TestDockerHost)
		r.Post("/test/coolify-host", ar.TestCoolifyHost)
		if ar.scanHandlers != nil {
			r.Get("/scan", ar.scanHandlers.GetScannerConfig)
			r.Group(func(mutating chi.Router) {
				mutating.Use(middleware.ReadOnly(func() bool {
					return ar.registry.Config().ReadOnly
				}))
				mutating.Put("/scan", ar.scanHandlers.UpdateScannerConfig)
				mutating.Post("/scan/test-notification", ar.scanHandlers.TestScanNotification)
			})
		}
	})
}

// handleLogin delegates to the dynamic auth service.
func (ar *APIRouter) handleLogin(w http.ResponseWriter, r *http.Request) {
	svc := ar.registry.Auth()
	if svc == nil || svc.IsDisabled() {
		http.Error(w, "Authentication is not enabled", http.StatusNotFound)
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if loginReq.Username == "" || loginReq.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	if err := svc.ValidateCredentials(loginReq.Username, loginReq.Password); err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	token, err := svc.GenerateToken(loginReq.Username)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"token": token,
		"user": map[string]string{
			"username": loginReq.Username,
			"role":     "admin",
		},
	})
}

// handleGetMe returns the current authenticated user's information.
func (ar *APIRouter) handleGetMe(w http.ResponseWriter, r *http.Request) {
	userValue := r.Context().Value(auth.UserContextKey)
	if userValue == nil {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"user": userValue,
	})
}
