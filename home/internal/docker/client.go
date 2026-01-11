package docker

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

type MultiHostClient struct {
	clients map[string]*client.Client
	hosts   []config.DockerHost
}

func NewMultiHostClient(hosts []config.DockerHost) (*MultiHostClient, error) {
	clients := make(map[string]*client.Client)

	for _, host := range hosts {
		var (
			apiClient *client.Client
			err       error
		)

		if strings.HasPrefix(host.Host, "ssh://") {
			helper, helperErr := connhelper.GetConnectionHelper(host.Host)
			if helperErr != nil {
				return nil, fmt.Errorf("failed to setup SSH helper for host %s (%s): %w", host.Name, host.Host, helperErr)
			}

			httpClient := &http.Client{
				Transport: &http.Transport{
					DialContext: helper.Dialer,
				},
			}

			apiClient, err = client.NewClientWithOpts(
				client.WithHTTPClient(httpClient),
				client.WithHost(helper.Host),
				client.WithDialContext(helper.Dialer),
				client.WithAPIVersionNegotiation(),
			)
		} else {
			apiClient, err = client.NewClientWithOpts(
				client.WithHost(host.Host),
				client.WithAPIVersionNegotiation(),
				client.FromEnv,
			)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to connect to host %s (%s): %w", host.Name, host.Host, err)
		}
		clients[host.Name] = apiClient
	}

	return &MultiHostClient{
		clients: clients,
		hosts:   hosts,
	}, nil
}

type HostError struct {
	HostName string
	Err      error
}

// hostResult holds the result of querying a single host
type hostResult struct {
	hostName   string
	containers []models.ContainerInfo
	err        error
}

func (c *MultiHostClient) ListContainersAllHosts(ctx context.Context) (map[string][]models.ContainerInfo, []HostError, error) {
	numHosts := len(c.clients)
	if numHosts == 0 {
		return make(map[string][]models.ContainerInfo), nil, nil
	}

	// Use channel to collect results from parallel queries
	resultCh := make(chan hostResult, numHosts)

	// Query all hosts in parallel
	var wg sync.WaitGroup
	for hostName, apiClient := range c.clients {
		wg.Add(1)
		go func(name string, client *client.Client) {
			defer wg.Done()
			c.queryHost(ctx, name, client, resultCh)
		}(hostName, apiClient)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	result := make(map[string][]models.ContainerInfo, numHosts)
	var hostErrors []HostError

	for hr := range resultCh {
		if hr.err != nil {
			hostErrors = append(hostErrors, HostError{HostName: hr.hostName, Err: hr.err})
			continue
		}
		result[hr.hostName] = hr.containers
	}

	return result, hostErrors, nil
}

// queryHost queries a single Docker host and sends result to channel
func (c *MultiHostClient) queryHost(ctx context.Context, hostName string, apiClient *client.Client, resultCh chan<- hostResult) {
	containers, err := apiClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		resultCh <- hostResult{hostName: hostName, err: err}
		return
	}

	hostContainers := make([]models.ContainerInfo, 0, len(containers))
	for _, ctr := range containers {
		hostContainers = append(hostContainers, models.ContainerInfo{
			ID:      ctr.ID,
			Names:   ctr.Names,
			Image:   ctr.Image,
			ImageID: ctr.ImageID,
			Command: ctr.Command,
			Created: ctr.Created,
			State:   ctr.State,
			Status:  ctr.Status,
			Labels:  ctr.Labels,
			Host:    hostName,
		})
	}

	resultCh <- hostResult{hostName: hostName, containers: hostContainers}
}

func (c *MultiHostClient) GetClient(hostName string) (*client.Client, error) {
	apiClient, ok := c.clients[hostName]
	if !ok {
		return nil, fmt.Errorf("host %s not found", hostName)
	}
	return apiClient, nil
}

func (c *MultiHostClient) GetHosts() []config.DockerHost {
	return c.hosts
}
