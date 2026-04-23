package services

import (
	"context"
	"errors"
	"sync"

	"github.com/hhftechnology/vps-monitor/internal/alerts"
	"github.com/hhftechnology/vps-monitor/internal/auth"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/coolify"
	"github.com/hhftechnology/vps-monitor/internal/docker"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// Registry holds all runtime services behind a RWMutex, allowing hot-swap.
type Registry struct {
	mu      sync.RWMutex
	docker  *docker.MultiHostClient
	coolify *coolify.MultiClient
	auth    *auth.Service
	config  *config.Config
	alerts  *alerts.Monitor

	dockerRefs    map[*docker.MultiHostClient]int
	dockerWaiters map[*docker.MultiHostClient]chan struct{}
}

// NewRegistry creates a registry with the initial set of services.
func NewRegistry(
	dockerClient *docker.MultiHostClient,
	coolifyClient *coolify.MultiClient,
	authService *auth.Service,
	cfg *config.Config,
	alertMonitor *alerts.Monitor,
) *Registry {
	return &Registry{
		docker:  dockerClient,
		coolify: coolifyClient,
		auth:    authService,
		config:  cfg,
		alerts:  alertMonitor,
		dockerRefs: func() map[*docker.MultiHostClient]int {
			refs := make(map[*docker.MultiHostClient]int)
			if dockerClient != nil {
				refs[dockerClient] = 0
			}
			return refs
		}(),
		dockerWaiters: make(map[*docker.MultiHostClient]chan struct{}),
	}
}

func (r *Registry) AcquireDocker() (*docker.MultiHostClient, func()) {
	r.mu.Lock()
	client := r.docker
	if client == nil {
		r.mu.Unlock()
		return nil, func() {}
	}
	r.dockerRefs[client]++
	r.mu.Unlock()

	var once sync.Once
	release := func() {
		once.Do(func() {
			r.mu.Lock()
			defer r.mu.Unlock()

			count, ok := r.dockerRefs[client]
			if !ok {
				return
			}
			count--
			if count <= 0 {
				delete(r.dockerRefs, client)
				if ch, waiting := r.dockerWaiters[client]; waiting {
					close(ch)
					delete(r.dockerWaiters, client)
				}
				return
			}
			r.dockerRefs[client] = count
		})
	}
	return client, release
}

func (r *Registry) StreamContainerStats(ctx context.Context, hostName, containerID string) (<-chan models.ContainerStats, <-chan error, error) {
	client, release := r.AcquireDocker()
	if client == nil {
		release()
		return nil, nil, errors.New("docker client unavailable")
	}

	srcStatsCh, srcErrCh := client.StreamContainerStats(ctx, hostName, containerID)
	statsCh := make(chan models.ContainerStats)
	errCh := make(chan error, 1)

	go func() {
		defer release()
		defer close(statsCh)
		defer close(errCh)

		for srcStatsCh != nil || srcErrCh != nil {
			select {
			case stat, ok := <-srcStatsCh:
				if !ok {
					srcStatsCh = nil
					continue
				}
				select {
				case statsCh <- stat:
				case <-ctx.Done():
					return
				}
			case err, ok := <-srcErrCh:
				if !ok {
					srcErrCh = nil
					continue
				}
				if err != nil {
					errCh <- err
				}
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return statsCh, errCh, nil
}

func (r *Registry) Coolify() *coolify.MultiClient {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.coolify
}

func (r *Registry) Auth() *auth.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.auth
}

func (r *Registry) Config() *config.Config {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.config
}

func (r *Registry) Alerts() *alerts.Monitor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.alerts
}

func (r *Registry) SwapDocker(newClient *docker.MultiHostClient) *docker.MultiHostClient {
	r.mu.Lock()
	old := r.docker
	if old == newClient {
		r.mu.Unlock()
		return old
	}
	r.docker = newClient
	if newClient != nil {
		if _, ok := r.dockerRefs[newClient]; !ok {
			r.dockerRefs[newClient] = 0
		}
	}

	if old == nil {
		r.mu.Unlock()
		return old
	}

	if refs := r.dockerRefs[old]; refs == 0 {
		delete(r.dockerRefs, old)
		r.mu.Unlock()
		old.Close()
		return old
	}

	waitCh := make(chan struct{})
	r.dockerWaiters[old] = waitCh
	r.mu.Unlock()

	<-waitCh
	old.Close()
	return old
}

func (r *Registry) SwapCoolify(newClient *coolify.MultiClient) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.coolify = newClient
}

func (r *Registry) SwapAuth(newService *auth.Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.auth = newService
}

func (r *Registry) UpdateConfig(cfg *config.Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.config = cfg
}

func (r *Registry) SwapAlerts(monitor *alerts.Monitor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alerts = monitor
}
