import type { ContainerInfo } from "../types";

export type SortDirection = "asc" | "desc";
export type GroupByOption = "none" | "compose";
export type ContainerActionType = "start" | "stop" | "restart" | "remove";

export interface GroupedContainers {
  project: string;
  items: ContainerInfo[];
}

export interface StateCounts {
  running: number;
  exited: number;
  paused: number;
  restarting: number;
  dead: number;
  other: number;
}

const COMPOSE_PROJECT_LABEL = "com.docker.compose.project";

export function formatContainerName(names: string[]) {
  if (!names.length) {
    return "—";
  }
  const [primary] = names;
  return primary.startsWith("/") ? primary.slice(1) : primary;
}

export function formatCreatedDate(createdSeconds: number) {
  const createdDate = new Date(createdSeconds * 1000);
  return createdDate.toLocaleString(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}

export function formatUptime(createdSeconds: number) {
  const now = Date.now();
  const createdMs = createdSeconds * 1000;
  const diffMs = now - createdMs;

  const seconds = Math.floor(diffMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  const weeks = Math.floor(days / 7);
  const months = Math.floor(days / 30);
  const years = Math.floor(days / 365);

  if (years > 0) return `${years} year${years > 1 ? "s" : ""}`;
  if (months > 0) return `${months} month${months > 1 ? "s" : ""}`;
  if (weeks > 0) return `${weeks} week${weeks > 1 ? "s" : ""}`;
  if (days > 0) return `${days} day${days > 1 ? "s" : ""}`;
  if (hours > 0) return `${hours} hour${hours > 1 ? "s" : ""}`;
  if (minutes > 0) return `${minutes} minute${minutes > 1 ? "s" : ""}`;
  return `${seconds} second${seconds !== 1 ? "s" : ""}`;
}

export function toTitleCase(value: string) {
  if (!value) return value;
  return value.charAt(0).toUpperCase() + value.slice(1);
}

export function getStateBadgeClass(state: string) {
  const normalized = state.toLowerCase();
  switch (normalized) {
    case "running":
      return "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400";
    case "paused":
      return "bg-amber-500/10 text-amber-700 dark:text-amber-400";
    case "exited":
    case "dead":
      return "bg-rose-500/10 text-rose-700 dark:text-rose-400";
    case "restarting":
      return "bg-blue-500/10 text-blue-700 dark:text-blue-400";
    default:
      return "bg-muted text-muted-foreground";
  }
}

export function groupByCompose(
  containers: ContainerInfo[]
): GroupedContainers[] {
  const groups = new Map<string, ContainerInfo[]>();

  containers.forEach((container) => {
    const key =
      container.labels?.[COMPOSE_PROJECT_LABEL]?.trim() || "Standalone";
    if (!groups.has(key)) {
      groups.set(key, []);
    }
    groups.get(key)?.push(container)
  });

  return Array.from(groups.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([project, items]) => ({ project, items }));
}

export function getInitialStateCounts(): StateCounts {
  return {
    running: 0,
    exited: 0,
    paused: 0,
    restarting: 0,
    dead: 0,
    other: 0,
  };
}

/**
 * Gets the container name for use in URLs (without leading slash)
 * Falls back to container ID if no name is available
 */
export function getContainerUrlIdentifier(container: ContainerInfo): string {
  if (container.names && container.names.length > 0) {
    const name = container.names[0];
    return name.startsWith("/") ? name.slice(1) : name;
  }
  // Fallback to short ID if no name
  return container.id.substring(0, 12);
}
