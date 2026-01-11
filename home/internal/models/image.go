package models

// ImageInfo represents a Docker image
type ImageInfo struct {
	ID          string            `json:"id"`
	RepoTags    []string          `json:"repo_tags"`
	RepoDigests []string          `json:"repo_digests,omitempty"`
	Size        int64             `json:"size"`
	VirtualSize int64             `json:"virtual_size,omitempty"`
	Created     int64             `json:"created"`
	Labels      map[string]string `json:"labels,omitempty"`
	Host        string            `json:"host"`
}

// ImagePullProgress represents progress during image pull
type ImagePullProgress struct {
	Status         string `json:"status"`
	Progress       string `json:"progress,omitempty"`
	ProgressDetail struct {
		Current int64 `json:"current,omitempty"`
		Total   int64 `json:"total,omitempty"`
	} `json:"progressDetail,omitempty"`
	ID    string `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

// ImageRemoveResult represents the result of removing an image
type ImageRemoveResult struct {
	Untagged []string `json:"untagged,omitempty"`
	Deleted  []string `json:"deleted,omitempty"`
}
