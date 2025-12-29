export interface ContainerStats {
  container_id: string;
  host: string;
  cpu_percent: number;
  memory_usage: number;
  memory_limit: number;
  memory_percent: number;
  network_rx: number;
  network_tx: number;
  block_read: number;
  block_write: number;
  pids: number;
  timestamp: number;
}
