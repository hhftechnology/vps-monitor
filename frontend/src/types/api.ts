/**
 * API Types for Docker Logs Viewer
 * Generated from backend API documentation
 */

// ============================================================================
// Container Types
// ============================================================================

export interface Container {
  id: string;
  names: string[];
  image: string;
  image_id: string;
  command: string;
  created: number; // Unix timestamp
  state: ContainerState;
  status: string; // Human-readable status
  labels?: Record<string, string>;
}

export type ContainerState =
  | "running"
  | "exited"
  | "paused"
  | "restarting"
  | "dead"
  | "created"
  | "removing";

// ============================================================================
// API Response Types
// ============================================================================

export interface ContainersResponse {
  containers: Container[];
  readOnly?: boolean;
}

export interface ContainerResponse {
  container: Container; // Note: Full InspectResponse from Docker
}

export interface ActionResponse {
  message: string;
}

// ============================================================================
// Log Options
// ============================================================================

export interface LogOptions {
  /** Stream logs in real-time (keep connection open) */
  follow?: boolean;
  /** Include timestamps for each log line */
  timestamps?: boolean;
  /** Number of lines to show from the end. Use "all" for all logs */
  tail?: string | number;
  /** Only return logs since this time (Unix timestamp or duration like "1h30m") */
  since?: string;
  /** Only return logs before this time (Unix timestamp or duration) */
  until?: string;
  /** Show extra log details */
  details?: boolean;
  /** Include stdout logs */
  stdout?: boolean;
  /** Include stderr logs */
  stderr?: boolean;
}

// ============================================================================
// API Client Configuration
// ============================================================================

// Use empty string for same-origin requests when frontend is served by backend
// This allows the browser to use relative paths (e.g., /api/v1/containers)
// Falls back to localhost:6789 for local development with separate servers
export const API_BASE_URL =
  import.meta.env.VITE_API_URL || "";

export const API_ENDPOINTS = {
  containers: {
    list: () => `${API_BASE_URL}/api/v1/containers`,
    get: (id: string) => `${API_BASE_URL}/api/v1/containers/${id}`,
    logs: (id: string) => `${API_BASE_URL}/api/v1/containers/${id}/logs`,
    start: (id: string) => `${API_BASE_URL}/api/v1/containers/${id}/start`,
    stop: (id: string) => `${API_BASE_URL}/api/v1/containers/${id}/stop`,
    restart: (id: string) => `${API_BASE_URL}/api/v1/containers/${id}/restart`,
    remove: (id: string) => `${API_BASE_URL}/api/v1/containers/${id}/remove`,
  },
} as const;

// ============================================================================
// Utility Types
// ============================================================================

/** Helper to build log query string */
export function buildLogQueryString(options: LogOptions): string {
  const params = new URLSearchParams();

  if (options.follow !== undefined)
    params.set("follow", String(options.follow));
  if (options.timestamps !== undefined)
    params.set("timestamps", String(options.timestamps));
  if (options.tail !== undefined) params.set("tail", String(options.tail));
  if (options.since) params.set("since", options.since);
  if (options.until) params.set("until", options.until);
  if (options.details !== undefined)
    params.set("details", String(options.details));
  if (options.stdout !== undefined)
    params.set("stdout", String(options.stdout));
  if (options.stderr !== undefined)
    params.set("stderr", String(options.stderr));

  return params.toString();
}

/** Get display-friendly container name (removes leading slash) */
export function getContainerDisplayName(container: Container): string {
  if (container.names.length === 0) return container.id.substring(0, 12);
  const name = container.names[0];
  return name.startsWith("/") ? name.substring(1) : name;
}

/** Get badge color for container state */
export function getStateColor(
  state: ContainerState
): "success" | "error" | "warning" | "default" {
  switch (state) {
    case "running":
      return "success";
    case "exited":
    case "dead":
      return "error";
    case "paused":
    case "restarting":
      return "warning";
    default:
      return "default";
  }
}

/** Format Unix timestamp to readable date */
export function formatContainerDate(timestamp: number): string {
  const date = new Date(timestamp * 1000);
  return date.toLocaleString();
}

/** Check if container is running */
export function isContainerRunning(container: Container): boolean {
  return container.state === "running";
}

/** Check if container can be started */
export function canStartContainer(container: Container): boolean {
  return container.state === "exited" || container.state === "created";
}

/** Check if container can be stopped */
export function canStopContainer(container: Container): boolean {
  return container.state === "running" || container.state === "paused";
}
