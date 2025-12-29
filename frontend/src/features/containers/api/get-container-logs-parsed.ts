import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

const BASE_URL = `${API_BASE_URL}/api/v1/containers`;

export type LogLevel =
  | "TRACE"
  | "DEBUG"
  | "INFO"
  | "WARN"
  | "WARNING"
  | "ERROR"
  | "FATAL"
  | "PANIC"
  | "UNKNOWN";

export interface LogEntry {
  timestamp?: string;
  level: LogLevel;
  message?: string;
  stream?: "stdout" | "stderr";
  raw?: string;
}

export interface ContainerLogsParsedResponse {
  logs: LogEntry[];
  count: number;
}

export interface ContainerLogsOptions {
  since?: string;
  until?: string;
  tail?: string | number;
  details?: boolean;
  stdout?: boolean;
  stderr?: boolean;
  follow?: boolean;
}

const DEFAULT_OPTIONS: Required<
  Pick<ContainerLogsOptions, "tail" | "details" | "stdout" | "stderr">
> = {
  tail: "100",
  details: false,
  stdout: true,
  stderr: true,
};

function buildLogsUrl(id: string, host: string, options?: ContainerLogsOptions) {
  const query = new URLSearchParams();
  const merged: ContainerLogsOptions = {
    ...DEFAULT_OPTIONS,
    ...options,
  };

  query.set("host", host);

  if (merged.since) {
    query.set("since", merged.since);
  }
  if (merged.until) {
    query.set("until", merged.until);
  }
  if (merged.tail !== undefined) {
    query.set("tail", String(merged.tail));
  }
  if (merged.details !== undefined) {
    query.set("details", String(merged.details));
  }
  if (merged.stdout !== undefined) {
    query.set("stdout", String(merged.stdout));
  }
  if (merged.stderr !== undefined) {
    query.set("stderr", String(merged.stderr));
  }
  if (merged.follow !== undefined) {
    query.set("follow", String(merged.follow));
  }

  const path = `${BASE_URL}/${encodeURIComponent(id)}/logs/parsed`;
  const queryString = query.toString();
  return queryString ? `${path}?${queryString}` : path;
}

export async function getContainerLogsParsed(
  id: string,
  host: string,
  options?: ContainerLogsOptions
): Promise<LogEntry[]> {
  const response = await authenticatedFetch(buildLogsUrl(id, host, options), {
    headers: {
      Accept: "application/json",
    },
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Failed to fetch logs for container ${id}`);
  }

  const data: ContainerLogsParsedResponse = await response.json();
  return data.logs || [];
}

async function* iterateNDJSONStream(
  stream: ReadableStream<Uint8Array>,
  signal?: AbortSignal
): AsyncGenerator<LogEntry, void, unknown> {
  const reader = stream.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      if (signal?.aborted) {
        reader.cancel().catch(() => {});
        break;
      }

      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      const lines = buffer.split("\n");
      buffer = lines.pop() ?? "";

      for (const line of lines) {
        if (line.trim()) {
          try {
            const entry: LogEntry = JSON.parse(line);
            yield entry;
          } catch (error) {
            console.error("Failed to parse log entry:", line, error);
          }
        }
      }
    }

    // Process remaining buffer
    if (buffer.trim()) {
      try {
        const entry: LogEntry = JSON.parse(buffer);
        yield entry;
      } catch (error) {
        console.error("Failed to parse final log entry:", buffer, error);
      }
    }
  } finally {
    reader.releaseLock();
  }
}

export async function* streamContainerLogsParsed(
  id: string,
  host: string,
  options?: ContainerLogsOptions,
  signal?: AbortSignal
): AsyncGenerator<LogEntry, void, unknown> {
  const streamOptions = { ...options, follow: true };
  const response = await authenticatedFetch(buildLogsUrl(id, host, streamOptions), {
    headers: {
      Accept: "application/x-ndjson",
    },
    signal,
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(
      message || `Failed to stream logs for container ${id}`
    );
  }

  if (!response.body) {
    throw new Error("Streaming is not supported in this environment.");
  }

  for await (const entry of iterateNDJSONStream(response.body, signal)) {
    yield entry;
  }
}

export function getLogLevelColor(level: LogLevel | undefined): string {
  switch (level ?? "UNKNOWN") {
    case "TRACE":
    case "DEBUG":
      return "text-muted-foreground";
    case "INFO":
      return "text-blue-600 dark:text-blue-400";
    case "WARN":
    case "WARNING":
      return "text-yellow-600 dark:text-yellow-400";
    case "ERROR":
      return "text-red-600 dark:text-red-400";
    case "FATAL":
    case "PANIC":
      return "text-red-700 dark:text-red-500 font-semibold";
    default:
      return "text-foreground";
  }
}

export function getLogLevelBadgeColor(level: LogLevel | undefined): string {
  switch (level ?? "UNKNOWN") {
    case "TRACE":
    case "DEBUG":
      return "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300";
    case "INFO":
      return "bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300";
    case "WARN":
    case "WARNING":
      return "bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300";
    case "ERROR":
      return "bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300";
    case "FATAL":
    case "PANIC":
      return "bg-red-200 text-red-900 dark:bg-red-950 dark:text-red-200 font-semibold";
    default:
      return "bg-muted text-muted-foreground";
  }
}
