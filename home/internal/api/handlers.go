package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hhftechnology/vps-monitor/internal/coolify"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/system"
)

// Pre-compiled regex for validating environment variable keys (performance optimization)
var envKeyRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func (ar *APIRouter) GetSystemStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	stats, err := system.GetStats(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Override hostname if configured
	cfg := ar.registry.Config()
	if cfg.Hostname != "" {
		stats.HostInfo.Hostname = cfg.Hostname
	}

	WriteJsonResponse(w, http.StatusOK, stats)
}

func (ar *APIRouter) GetContainers(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	containersMap, hostErrors, err := ar.registry.Docker().ListContainersAllHosts(ctx)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Flatten the map for easier frontend consumption
	allContainers := []models.ContainerInfo{}
	for _, containers := range containersMap {
		allContainers = append(allContainers, containers...)
	}

	// Build host errors list for the frontend (graceful partial results)
	hostErrorMessages := make([]map[string]string, 0, len(hostErrors))
	for _, he := range hostErrors {
		hostErrorMessages = append(hostErrorMessages, map[string]string{
			"host":    he.HostName,
			"message": he.Err.Error(),
		})
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"containers":        allContainers,
		"hosts":             ar.registry.Docker().GetHosts(),
		"readOnly":          ar.registry.Config().ReadOnly,
		"hostErrors":        hostErrorMessages,
		"coolifyConfigured": ar.registry.Coolify() != nil,
	})
}

func (ar *APIRouter) GetContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	inspect, err := ar.registry.Docker().GetContainer(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name := inspect.Name
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}

	var command string
	if inspect.Config != nil && len(inspect.Config.Cmd) > 0 {
		command = fmt.Sprintf("%v", inspect.Config.Cmd)
	}

	var image string
	if inspect.Config != nil {
		image = inspect.Config.Image
	}

	var labels map[string]string
	if inspect.Config != nil {
		labels = inspect.Config.Labels
	}

	state := ""
	status := ""
	if inspect.State != nil {
		state = inspect.State.Status
		status = inspect.State.Status
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"container": map[string]any{
			"id":      inspect.ID,
			"names":   []string{name},
			"image":   image,
			"state":   state,
			"status":  status,
			"host":    host,
			"created": inspect.Created,
			"command": command,
			"labels":  labels,
		},
	})
}

func (ar *APIRouter) StartContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	err := ar.registry.Docker().StartContainer(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "Container started",
	})
}

func (ar *APIRouter) StopContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	err := ar.registry.Docker().StopContainer(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "Container stopped",
	})
}

func (ar *APIRouter) RestartContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	err := ar.registry.Docker().RestartContainer(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "Container restarted",
	})
}

func (ar *APIRouter) RemoveContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	err := ar.registry.Docker().RemoveContainer(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"message": "Container removed",
	})
}

func (ar *APIRouter) GetContainerLogsParsed(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	options := parseLogOptions(r)

	if options.Follow {
		ar.streamParsedLogs(w, host, id, options)
		return
	}

	logs, err := ar.registry.Docker().GetContainerLogsParsed(host, id, options)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"logs":  logs,
		"count": len(logs),
	})
}

func (ar *APIRouter) streamParsedLogs(w http.ResponseWriter, host, id string, options models.LogOptions) {
	stream, err := ar.registry.Docker().StreamContainerLogsParsed(host, id, options)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	buffer := make([]byte, 32*1024)
	for {
		n, readErr := stream.Read(buffer)
		if n > 0 {
			if _, writeErr := w.Write(buffer[:n]); writeErr != nil {
				break
			}
			flusher.Flush()
		}
		if readErr != nil {
			break
		}
	}
}

func parseLogOptions(r *http.Request) models.LogOptions {
	query := r.URL.Query()

	options := models.DefaultLogOptions()

	if follow := query.Get("follow"); follow != "" {
		options.Follow, _ = strconv.ParseBool(follow)
	}

	if timestamps := query.Get("timestamps"); timestamps != "" {
		options.Timestamps, _ = strconv.ParseBool(timestamps)
	}

	if since := query.Get("since"); since != "" {
		options.Since = since
	}

	if until := query.Get("until"); until != "" {
		options.Until = until
	}

	if tail := query.Get("tail"); tail != "" {
		options.Tail = tail
	}

	if details := query.Get("details"); details != "" {
		options.Details, _ = strconv.ParseBool(details)
	}

	if stdout := query.Get("stdout"); stdout != "" {
		options.ShowStdout, _ = strconv.ParseBool(stdout)
	}

	if stderr := query.Get("stderr"); stderr != "" {
		options.ShowStderr, _ = strconv.ParseBool(stderr)
	}

	return options
}

func (ar *APIRouter) GetEnvVariables(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	envVariables, err := ar.registry.Docker().GetEnvVariables(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"env": envVariables,
	})
}

func (ar *APIRouter) UpdateEnvVariables(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	var envVariables models.EnvVariables
	if err := json.NewDecoder(r.Body).Decode(&envVariables); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for key := range envVariables.Env {
		if !envKeyRegex.MatchString(key) {
			http.Error(w, fmt.Sprintf("invalid environment variable key: %s", key), http.StatusBadRequest)
			return
		}
	}

	newContainerID, labels, err := ar.registry.Docker().SetEnvVariables(r.Context(), host, id, envVariables.Env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"message":          "Environment variables updated",
		"new_container_id": newContainerID,
	}

	// Best-effort sync to Coolify API
	coolifyMulti := ar.registry.Coolify()
	if coolifyMulti != nil {
		coolifyClient := coolifyMulti.GetClient(host)
		coolifyResource := coolify.ExtractResourceInfo(labels)
		if coolifyClient != nil && coolifyResource != nil {
			syncCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()

			if syncErr := coolifyClient.SyncEnvVars(syncCtx, coolifyResource, envVariables.Env); syncErr != nil {
				log.Printf("Warning: failed to sync env vars to Coolify for host %s: %v", host, syncErr)
				response["coolify_synced"] = false
				response["coolify_error"] = syncErr.Error()
			} else {
				response["coolify_synced"] = true
			}
		}
	}

	WriteJsonResponse(w, http.StatusOK, response)
}
