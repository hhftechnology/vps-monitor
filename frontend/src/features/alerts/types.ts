export type AlertType =
  | "container_stopped"
  | "cpu_threshold"
  | "memory_threshold";

export interface Alert {
  id: string;
  type: AlertType;
  container_id: string;
  container_name: string;
  host: string;
  message: string;
  value?: number;
  threshold?: number;
  timestamp: number;
  acknowledged: boolean;
}

export interface AlertConfig {
  enabled: boolean;
  cpu_threshold: number;
  memory_threshold: number;
  check_interval: string;
  webhook_configured: boolean;
}
