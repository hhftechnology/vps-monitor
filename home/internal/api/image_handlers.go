package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// GetImages lists all images across all Docker hosts
func (ar *APIRouter) GetImages(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	imagesMap, hostErrors, err := ar.docker.ListImagesAllHosts(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(hostErrors) > 0 {
		http.Error(w, fmt.Sprintf("Error listing images on some hosts: %v", hostErrors), http.StatusInternalServerError)
		return
	}

	// Flatten the map for easier frontend consumption
	allImages := []models.ImageInfo{}
	for _, images := range imagesMap {
		allImages = append(allImages, images...)
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"images":   allImages,
		"hosts":    ar.docker.GetHosts(),
		"readOnly": ar.config.ReadOnly,
	})
}

// GetImage returns details of a specific image
func (ar *APIRouter) GetImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	image, err := ar.docker.GetImage(r.Context(), host, id)
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
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")
	forceStr := r.URL.Query().Get("force")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	force, _ := strconv.ParseBool(forceStr)

	result, err := ar.docker.RemoveImage(r.Context(), host, id, force)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	reader, err := ar.docker.PullImage(r.Context(), host, imageName)
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
