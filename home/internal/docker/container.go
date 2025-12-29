package docker

import (
	"context"
	"maps"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

func (c *MultiHostClient) GetContainer(ctx context.Context, hostName, id string) (container.InspectResponse, error) {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return container.InspectResponse{}, err
	}
	result, err := apiClient.ContainerInspect(ctx, id)
	if err != nil {
		return container.InspectResponse{}, err
	}
	return result, nil
}

func (c *MultiHostClient) StartContainer(ctx context.Context, hostName, id string) error {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return err
	}
	return apiClient.ContainerStart(ctx, id, container.StartOptions{})
}

func (c *MultiHostClient) StopContainer(ctx context.Context, hostName, id string) error {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return err
	}
	return apiClient.ContainerStop(ctx, id, container.StopOptions{})
}

func (c *MultiHostClient) RestartContainer(ctx context.Context, hostName, id string) error {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return err
	}
	return apiClient.ContainerRestart(ctx, id, container.StopOptions{})
}

func (c *MultiHostClient) RemoveContainer(ctx context.Context, hostName, id string) error {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return err
	}
	return apiClient.ContainerRemove(ctx, id, container.RemoveOptions{})
}

func (c *MultiHostClient) GetEnvVariables(ctx context.Context, hostName, id string) (map[string]string, error) {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return nil, err
	}

	inspect, err := apiClient.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	envMap := make(map[string]string)
	for _, env := range inspect.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	return envMap, nil
}

func (c *MultiHostClient) SetEnvVariables(ctx context.Context, hostName, id string, envVariables map[string]string) (string, error) {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return "", err
	}

	inspect, err := apiClient.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}

	envMap := make(map[string]string)
	// First, load all existing env vars from the container config
	for _, env := range inspect.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Delete keys from envMap that are not present in envVariables
	for key := range envMap {
		if _, exists := envVariables[key]; !exists {
			delete(envMap, key)
		}
	}

	// Copy updated/new variables from envVariables into envMap
	maps.Copy(envMap, envVariables)

	envs := make([]string, 0, len(envMap))
	for key, value := range envMap {
		envs = append(envs, key+"="+value)
	}

	containerName := inspect.Name
	containerName = strings.TrimPrefix(containerName, "/")

	err = apiClient.ContainerStop(ctx, id, container.StopOptions{})
	if err != nil {
		return "", err
	}

	err = apiClient.ContainerRemove(ctx, id, container.RemoveOptions{})
	if err != nil {
		return "", err
	}

	newConfig := inspect.Config
	newConfig.Env = envs

	resp, err := apiClient.ContainerCreate(
		ctx,
		newConfig,
		inspect.HostConfig,
		&network.NetworkingConfig{
			EndpointsConfig: inspect.NetworkSettings.Networks,
		},
		nil,
		containerName,
	)
	if err != nil {
		return "", err
	}

	err = apiClient.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}
