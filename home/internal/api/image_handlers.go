package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// GetImages lists all images across all Docker hosts
func (ar *APIRouter) GetImages(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	defer releaseDocker()
	if dockerClient == nil {
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	imagesMap, hostErrors, err := dockerClient.ListImagesAllHosts(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Flatten the map for easier frontend consumption
	allImages := []models.ImageInfo{}
	for _, images := range imagesMap {
		allImages = append(allImages, images...)
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
		"images":     allImages,
		"hosts":      dockerClient.GetHosts(),
		"readOnly":   ar.registry.Config().ReadOnly,
		"hostErrors": hostErrorMessages,
	})
}

// GetImage returns details of a specific image
func (ar *APIRouter) GetImage(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	// URL-decode the image ID (handles sha256%3A... -> sha256:...)
	id, err := url.PathUnescape(rawID)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid image ID: %v", err), http.StatusBadRequest)
		return
	}

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	defer releaseDocker()
	if dockerClient == nil {
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	image, err := dockerClient.GetImage(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"image": image,
	})
}

// RemoveImage removes an image from a host
func (ar *APIRouter) RemoveImage(w http.ResponseWriter, r *http.Request) {
	rawID := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")
	forceStr := r.URL.Query().Get("force")

	// URL-decode the image ID (handles sha256%3A... -> sha256:...)
	id, err := url.PathUnescape(rawID)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid image ID: %v", err), http.StatusBadRequest)
		return
	}

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	force, _ := strconv.ParseBool(forceStr)

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	defer releaseDocker()
	if dockerClient == nil {
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	result, err := dockerClient.RemoveImage(r.Context(), host, id, force)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove image: %v", err), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "Image removed",
		"result":  result,
	})
}

// PullImage pulls an image and streams progress as NDJSON
func (ar *APIRouter) PullImage(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("host")
	imageName := r.URL.Query().Get("image")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	if imageName == "" {
		http.Error(w, "image parameter is required", http.StatusBadRequest)
		return
	}

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	defer releaseDocker()
	if dockerClient == nil {
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	reader, err := dockerClient.PullImage(r.Context(), host, imageName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	decoder := json.NewDecoder(reader)
	encoder := json.NewEncoder(w)

	for {
		var progress models.ImagePullProgress
		if err := decoder.Decode(&progress); err != nil {
			if err == io.EOF {
				break
			}
			// Send error in stream
			_ = encoder.Encode(models.ImagePullProgress{
				Status: "error",
				Error:  err.Error(),
			})
			break
		}

		if err := encoder.Encode(progress); err != nil {
			break
		}
		flusher.Flush()
	}

	// Send completion message
	_ = encoder.Encode(models.ImagePullProgress{
		Status: "complete",
	})
	flusher.Flush()
}
