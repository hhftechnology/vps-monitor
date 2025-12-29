package models

// NetworkInfo represents a Docker network
type NetworkInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope"`
	Internal   bool              `json:"internal"`
	EnableIPv6 bool              `json:"enable_ipv6"`
	Labels     map[string]string `json:"labels,omitempty"`
	Host       string            `json:"host"`
	Containers int               `json:"containers"` // Count of connected containers
}

// NetworkDetails represents detailed network information
type NetworkDetails struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Driver     string             `json:"driver"`
	Scope      string             `json:"scope"`
	Internal   bool               `json:"internal"`
	EnableIPv6 bool               `json:"enable_ipv6"`
	Labels     map[string]string  `json:"labels,omitempty"`
	Host       string             `json:"host"`
	IPAM       IPAMConfig         `json:"ipam"`
	Containers []NetworkContainer `json:"connected_containers"`
	Options    map[string]string  `json:"options,omitempty"`
	Created    string             `json:"created"`
}

// IPAMConfig represents IP Address Management configuration
type IPAMConfig struct {
	Driver  string     `json:"driver"`
	Config  []IPAMPool `json:"config"`
	Options map[string]string `json:"options,omitempty"`
}

// IPAMPool represents an IP address pool
type IPAMPool struct {
	Subnet     string `json:"subnet"`
	Gateway    string `json:"gateway,omitempty"`
	IPRange    string `json:"ip_range,omitempty"`
	AuxAddress map[string]string `json:"aux_addresses,omitempty"`
}

// NetworkContainer represents a container connected to a network
type NetworkContainer struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	IPv4Address   string `json:"ipv4_address"`
	IPv6Address   string `json:"ipv6_address,omitempty"`
	MacAddress    string `json:"mac_address"`
}
