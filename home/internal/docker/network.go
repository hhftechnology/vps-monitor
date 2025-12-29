package docker

import (
	"context"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/network"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// networkResult holds the result of querying networks from a single host
type networkResult struct {
	hostName string
	networks []models.NetworkInfo
	err      error
}

// ListNetworksAllHosts lists networks across all Docker hosts in parallel
func (c *MultiHostClient) ListNetworksAllHosts(ctx context.Context) (map[string][]models.NetworkInfo, []HostError, error) {
	numHosts := len(c.clients)
	if numHosts == 0 {
		return make(map[string][]models.NetworkInfo), nil, nil
	}

	resultCh := make(chan networkResult, numHosts)

	var wg sync.WaitGroup
	for hostName, apiClient := range c.clients {
		wg.Add(1)
		go func(name string, client networkLister) {
			defer wg.Done()
			c.queryNetworks(ctx, name, client, resultCh)
		}(hostName, apiClient)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	result := make(map[string][]models.NetworkInfo, numHosts)
	var hostErrors []HostError

	for nr := range resultCh {
		if nr.err != nil {
			hostErrors = append(hostErrors, HostError{HostName: nr.hostName, Err: nr.err})
			continue
		}
		result[nr.hostName] = nr.networks
	}

	return result, hostErrors, nil
}

// networkLister interface for testing
type networkLister interface {
	NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error)
}

// queryNetworks queries networks from a single Docker host
func (c *MultiHostClient) queryNetworks(ctx context.Context, hostName string, apiClient networkLister, resultCh chan<- networkResult) {
	networks, err := apiClient.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		resultCh <- networkResult{hostName: hostName, err: err}
		return
	}

	hostNetworks := make([]models.NetworkInfo, 0, len(networks))
	for _, net := range networks {
		hostNetworks = append(hostNetworks, models.NetworkInfo{
			ID:         net.ID,
			Name:       net.Name,
			Driver:     net.Driver,
			Scope:      net.Scope,
			Internal:   net.Internal,
			EnableIPv6: net.EnableIPv6,
			Labels:     net.Labels,
			Host:       hostName,
			Containers: len(net.Containers),
		})
	}

	resultCh <- networkResult{hostName: hostName, networks: hostNetworks}
}

// GetNetworkDetails returns detailed information about a network
func (c *MultiHostClient) GetNetworkDetails(ctx context.Context, hostName, networkID string) (*models.NetworkDetails, error) {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return nil, err
	}

	net, err := apiClient.NetworkInspect(ctx, networkID, network.InspectOptions{Verbose: true})
	if err != nil {
		return nil, err
	}

	// Build IPAM config
	ipam := models.IPAMConfig{
		Driver:  net.IPAM.Driver,
		Options: net.IPAM.Options,
	}
	for _, cfg := range net.IPAM.Config {
		ipam.Config = append(ipam.Config, models.IPAMPool{
			Subnet:  cfg.Subnet,
			Gateway: cfg.Gateway,
			IPRange: cfg.IPRange,
		})
	}

	// Build connected containers list
	containers := make([]models.NetworkContainer, 0, len(net.Containers))
	for id, ctr := range net.Containers {
		containers = append(containers, models.NetworkContainer{
			ContainerID:   id,
			ContainerName: strings.TrimPrefix(ctr.Name, "/"),
			IPv4Address:   ctr.IPv4Address,
			IPv6Address:   ctr.IPv6Address,
			MacAddress:    ctr.MacAddress,
		})
	}

	return &models.NetworkDetails{
		ID:         net.ID,
		Name:       net.Name,
		Driver:     net.Driver,
		Scope:      net.Scope,
		Internal:   net.Internal,
		EnableIPv6: net.EnableIPv6,
		Labels:     net.Labels,
		Host:       hostName,
		IPAM:       ipam,
		Containers: containers,
		Options:    net.Options,
		Created:    net.Created.String(),
	}, nil
}
