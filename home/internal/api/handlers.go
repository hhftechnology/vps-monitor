package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hhftechnology/vps-monitor/internal/coolify"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/system"
	"github.com/hhftechnology/vps-monitor/internal/docker"
	"sync"
)

// Pre-compiled regex for validating environment variable keys (performance optimization)
var envKeyRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

const containerStatsBootstrapLimit = 60

type ContainerActionJob struct {
	Host      string
	ID        string
	Action    string
	Status    string
	Error     string
	ExpiresAt time.Time
}

var (
	actionJobsMu sync.RWMutex
	actionJobs   = make(map[string]*ContainerActionJob)
)

func RecordActionJob(host, id, action, status, errStr string) {
	actionJobsMu.Lock()
	defer actionJobsMu.Unlock()
	key := host + ":" + id + ":" + action
	actionJobs[key] = &ContainerActionJob{
		Host:      host,
		ID:        id,
		Action:    action,
		Status:    status,
		Error:     errStr,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	for k, v := range actionJobs {
		if time.Now().After(v.ExpiresAt) {
			delete(actionJobs, k)
		}
	}
}

func GetActionJob(host, id, action string) *ContainerActionJob {
	actionJobsMu.RLock()
	defer actionJobsMu.RUnlock()
	return actionJobs[host+":"+id+":"+action]
}

type coolifyEnvSyncer interface {
	SyncEnvVars(ctx context.Context, resource *coolify.ResourceInfo, envVars map[string]string) error
}

func isNilCoolifySyncer(syncer coolifyEnvSyncer) bool {
	if syncer == nil {
		return true
	}
	value := reflect.ValueOf(syncer)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (ar *APIRouter) getContainerHistoricalAverages(host, containerID string) (models.HistoricalAverages, error) {
	if ar.statsDB == nil {
		return models.HistoricalAverages{}, fmt.Errorf("stats database not available")
	}
	return ar.statsDB.GetContainerHistoricalAverages(host, containerID, time.Now())
}

func (ar *APIRouter) getContainerHistoricalSamples(host, containerID string) ([]models.ContainerStats, error) {
	if ar.statsDB == nil {
		return nil, fmt.Errorf("stats database not available")
	}
	return ar.statsDB.GetRecentContainerStats(
		host,
		containerID,
		time.Now().Add(-12*time.Hour),
		containerStatsBootstrapLimit,
	)
}

func (ar *APIRouter) enrichContainersWithHistoricalStats(containers []models.ContainerInfo) {
	if ar.statsDB == nil {
		return
	}

	for i := range containers {
		history, err := ar.getContainerHistoricalAverages(containers[i].Host, containers[i].ID)
		if err != nil {
			log.Printf("failed to load container history for %s on %s: %v", containers[i].ID, containers[i].Host, err)
			continue
		}
		if history.HasData {
			containers[i].HistoricalStats = &models.HistoricalStats{
				CPU1h:     history.CPU1h,
				Memory1h:  history.Memory1h,
				CPU12h:    history.CPU12h,
				Memory12h: history.Memory12h,
			}
		}
	}
}

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
	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	if dockerClient == nil {
		releaseDocker()
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}
	containersMap, hostErrors, err := dockerClient.ListContainersAllHosts(ctx)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Flatten the map for easier frontend consumption
	allContainers := []models.ContainerInfo{}
	for _, containers := range containersMap {
		allContainers = append(allContainers, containers...)
	}

	ar.enrichContainersWithHistoricalStats(allContainers)

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
		"hosts":             dockerClient.GetHosts(),
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

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	defer releaseDocker()
	if dockerClient == nil {
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	inspect, err := dockerClient.GetContainer(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name := inspect.Name
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}

	command := ""
	if inspect.Config != nil && len(inspect.Config.Cmd) > 0 {
		command = strings.Join(inspect.Config.Cmd, " ")
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

	names := make([]string, 0, 1)
	if name != "" {
		names = append(names, name)
	}

	createdAt := int64(0)
	if inspect.Created != "" {
		if parsedCreated, parseErr := time.Parse(time.RFC3339Nano, inspect.Created); parseErr == nil {
			createdAt = parsedCreated.Unix()
		}
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"container": map[string]any{
			"id":       inspect.ID,
			"names":    names,
			"image":    image,
			"image_id": inspect.Image,
			"state":    state,
			"status":   status,
			"host":     host,
			"created":  createdAt,
			"command":  command,
			"labels":   labels,
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

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	defer releaseDocker()
	if dockerClient == nil {
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	err := dockerClient.StartContainer(r.Context(), host, id)
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

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	defer releaseDocker()
	if dockerClient == nil {
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	WriteJsonResponse(w, http.StatusAccepted, map[string]any{
		"message": "Container stop initiated",
		"status":  "pending",
	})

	ar.runAsyncContainerAction(host, id, "stop", func(ctx context.Context, client *docker.MultiHostClient) error {
		return client.StopContainer(ctx, host, id)
	})
}

func (ar *APIRouter) RestartContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	if dockerClient == nil {
		releaseDocker()
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	WriteJsonResponse(w, http.StatusAccepted, map[string]any{
		"message": "Container restart initiated",
		"status":  "pending",
	})

	ar.runAsyncContainerAction(host, id, "restart", func(ctx context.Context, client *docker.MultiHostClient) error {
		return client.RestartContainer(ctx, host, id)
	})
}

func (ar *APIRouter) RemoveContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	if dockerClient == nil {
		releaseDocker()
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	WriteJsonResponse(w, http.StatusAccepted, map[string]any{
		"message": "Container remove initiated",
		"status":  "pending",
	})

	ar.runAsyncContainerAction(host, id, "remove", func(ctx context.Context, client *docker.MultiHostClient) error {
		return client.RemoveContainer(ctx, host, id)
	})
}

func (ar *APIRouter) GetContainerHistoricalStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	if ar.statsDB == nil {
		http.Error(w, "stats history not available", http.StatusServiceUnavailable)
		return
	}

	history, err := ar.getContainerHistoricalAverages(host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	samples, err := ar.getContainerHistoricalSamples(host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	history.Samples = samples
	WriteJsonResponse(w, http.StatusOK, history)
}

func (ar *APIRouter) runAsyncContainerAction(host, id, action string, fn func(context.Context, *docker.MultiHostClient) error) {
	RecordActionJob(host, id, action, "pending", "")
	go func() {
		dockerClient, release := ar.registry.AcquireDocker()
		defer release()

		if dockerClient == nil {
			log.Printf("failed to %s container %s on host %s: docker client unavailable", action, id, host)
			RecordActionJob(host, id, action, "failed", "docker client unavailable")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		if err := fn(ctx, dockerClient); err != nil {
			log.Printf("failed to %s container %s on host %s: %v", action, id, host, err)
			RecordActionJob(host, id, action, "failed", err.Error())
		} else {
			RecordActionJob(host, id, action, "success", "")
		}
	}()
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

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	defer releaseDocker()
	if dockerClient == nil {
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	logs, err := dockerClient.GetContainerLogsParsed(host, id, options)
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
	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	defer releaseDocker()
	if dockerClient == nil {
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	stream, err := dockerClient.StreamContainerLogsParsed(host, id, options)
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

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	defer releaseDocker()
	if dockerClient == nil {
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	envVariables, err := dockerClient.GetEnvVariables(r.Context(), host, id)
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

	dockerClient, releaseDocker := ar.registry.AcquireDocker()
	defer releaseDocker()
	if dockerClient == nil {
		http.Error(w, "docker client unavailable", http.StatusServiceUnavailable)
		return
	}

	newContainerID, labels, err := dockerClient.SetEnvVariables(r.Context(), host, id, envVariables.Env)
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
		if isNilCoolifySyncer(coolifyClient) {
			log.Printf("Warning: Coolify client unavailable for host %s; skipping env sync", host)
		} else {
			coolifyResource := coolify.ExtractResourceInfo(labels)
			applyCoolifyEnvSync(r.Context(), host, coolifyClient, coolifyResource, envVariables.Env, response)
		}
	}

	WriteJsonResponse(w, http.StatusOK, response)
}

func applyCoolifyEnvSync(ctx context.Context, host string, syncer coolifyEnvSyncer, resource *coolify.ResourceInfo, env map[string]string, response map[string]any) {
	if isNilCoolifySyncer(syncer) || resource == nil {
		return
	}
	if resource.Type == coolify.ResourceTypeDatabase {
		response["coolify_synced"] = false
		response["coolify_error"] = "sync not supported for database resources"
		return
	}

	syncCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if syncErr := syncer.SyncEnvVars(syncCtx, resource, env); syncErr != nil {
		log.Printf("Warning: failed to sync env vars to Coolify for host %s: %v", host, syncErr)
		response["coolify_synced"] = false
		response["coolify_error"] = syncErr.Error()
		return
	}
	response["coolify_synced"] = true
}
