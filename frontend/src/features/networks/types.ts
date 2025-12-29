export interface NetworkInfo {
  id: string;
  name: string;
  driver: string;
  scope: string;
  internal: boolean;
  enable_ipv6: boolean;
  labels: Record<string, string> | null;
  host: string;
  containers: number;
}

export interface IPAMPool {
  subnet: string;
  gateway: string;
  ip_range: string;
}

export interface IPAMConfig {
  driver: string;
  options: Record<string, string> | null;
  config: IPAMPool[];
}

export interface NetworkContainer {
  container_id: string;
  container_name: string;
  ipv4_address: string;
  ipv6_address: string;
  mac_address: string;
}

export interface NetworkDetails {
  id: string;
  name: string;
  driver: string;
  scope: string;
  internal: boolean;
  enable_ipv6: boolean;
  labels: Record<string, string> | null;
  host: string;
  ipam: IPAMConfig;
  connected_containers: NetworkContainer[];
  options: Record<string, string> | null;
  created: string;
}
