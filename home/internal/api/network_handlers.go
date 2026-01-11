package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// GetNetworks lists all networks across all Docker hosts
func (ar *APIRouter) GetNetworks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	networksMap, hostErrors, err := ar.docker.ListNetworksAllHosts(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(hostErrors) > 0 {
		http.Error(w, fmt.Sprintf("Error listing networks on some hosts: %v", hostErrors), http.StatusInternalServerError)
		return
	}

	// Flatten the map for easier frontend consumption
	allNetworks := []models.NetworkInfo{}
	for _, networks := range networksMap {
		allNetworks = append(allNetworks, networks...)
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"networks": allNetworks,
		"hosts":    ar.docker.GetHosts(),
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

	network, err := ar.docker.GetNetworkDetails(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"network": network,
	})
}
