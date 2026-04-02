package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// GetNetworks lists all networks across all Docker hosts
func (ar *APIRouter) GetNetworks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	networksMap, hostErrors, err := ar.registry.Docker().ListNetworksAllHosts(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Flatten the map for easier frontend consumption
	allNetworks := []models.NetworkInfo{}
	for _, networks := range networksMap {
		allNetworks = append(allNetworks, networks...)
	}

	// Build host errors list (graceful partial results)
	hostErrorMessages := make([]map[string]string, 0, len(hostErrors))
	for _, he := range hostErrors {
		hostErrorMessages = append(hostErrorMessages, map[string]string{
			"host":    he.HostName,
			"message": he.Err.Error(),
		})
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"networks":   allNetworks,
		"hosts":      ar.registry.Docker().GetHosts(),
		"hostErrors": hostErrorMessages,
	})
}

// GetNetwork returns detailed information about a specific network
func (ar *APIRouter) GetNetwork(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	network, err := ar.registry.Docker().GetNetworkDetails(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"network": network,
	})
}
