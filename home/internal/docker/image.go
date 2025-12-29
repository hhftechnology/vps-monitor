package docker

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/docker/docker/api/types/image"
	"github.com/hhftechnology/vps-monitor/internal/models"
)

// imageResult holds the result of querying images from a single host
type imageResult struct {
	hostName string
	images   []models.ImageInfo
	err      error
}

// ListImagesAllHosts lists images across all Docker hosts in parallel
func (c *MultiHostClient) ListImagesAllHosts(ctx context.Context) (map[string][]models.ImageInfo, []HostError, error) {
	numHosts := len(c.clients)
	if numHosts == 0 {
		return make(map[string][]models.ImageInfo), nil, nil
	}

	resultCh := make(chan imageResult, numHosts)

	var wg sync.WaitGroup
	for hostName, apiClient := range c.clients {
		wg.Add(1)
		go func(name string, client dockerClient) {
			defer wg.Done()
			c.queryImages(ctx, name, client, resultCh)
		}(hostName, apiClient)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	result := make(map[string][]models.ImageInfo, numHosts)
	var hostErrors []HostError

	for ir := range resultCh {
		if ir.err != nil {
			hostErrors = append(hostErrors, HostError{HostName: ir.hostName, Err: ir.err})
			continue
		}
		result[ir.hostName] = ir.images
	}

	return result, hostErrors, nil
}

// dockerClient interface for testing
type dockerClient interface {
	ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
}

// queryImages queries images from a single Docker host
func (c *MultiHostClient) queryImages(ctx context.Context, hostName string, apiClient dockerClient, resultCh chan<- imageResult) {
	images, err := apiClient.ImageList(ctx, image.ListOptions{All: false})
	if err != nil {
		resultCh <- imageResult{hostName: hostName, err: err}
		return
	}

	hostImages := make([]models.ImageInfo, 0, len(images))
	for _, img := range images {
		hostImages = append(hostImages, models.ImageInfo{
			ID:          img.ID,
			RepoTags:    img.RepoTags,
			RepoDigests: img.RepoDigests,
			Size:        img.Size,
			VirtualSize: img.VirtualSize,
			Created:     img.Created,
			Labels:      img.Labels,
			Host:        hostName,
		})
	}

	resultCh <- imageResult{hostName: hostName, images: hostImages}
}

// GetImage returns details of a specific image
func (c *MultiHostClient) GetImage(ctx context.Context, hostName, imageID string) (*models.ImageInfo, error) {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return nil, err
	}

	inspect, _, err := apiClient.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return nil, err
	}

	// Parse created time from string
	var createdUnix int64
	if t, err := time.Parse(time.RFC3339, inspect.Created); err == nil {
		createdUnix = t.Unix()
	}

	var labels map[string]string
	if inspect.Config != nil {
		labels = inspect.Config.Labels
	}

	return &models.ImageInfo{
		ID:          inspect.ID,
		RepoTags:    inspect.RepoTags,
		RepoDigests: inspect.RepoDigests,
		Size:        inspect.Size,
		VirtualSize: inspect.VirtualSize,
		Created:     createdUnix,
		Labels:      labels,
		Host:        hostName,
	}, nil
}

// RemoveImage removes an image from a host
func (c *MultiHostClient) RemoveImage(ctx context.Context, hostName, imageID string, force bool) (*models.ImageRemoveResult, error) {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return nil, err
	}

	responses, err := apiClient.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
	if err != nil {
		return nil, err
	}

	result := &models.ImageRemoveResult{}
	for _, resp := range responses {
		if resp.Untagged != "" {
			result.Untagged = append(result.Untagged, resp.Untagged)
		}
		if resp.Deleted != "" {
			result.Deleted = append(result.Deleted, resp.Deleted)
		}
	}

	return result, nil
}

// PullImage pulls an image and returns a reader for progress
func (c *MultiHostClient) PullImage(ctx context.Context, hostName, imageName string) (io.ReadCloser, error) {
	apiClient, err := c.GetClient(hostName)
	if err != nil {
		return nil, err
	}

	reader, err := apiClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return nil, err
	}

	return reader, nil
}
