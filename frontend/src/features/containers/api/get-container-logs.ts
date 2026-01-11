import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

const BASE_URL = `${API_BASE_URL}/api/v1/containers`;

export interface ContainerLogsOptions {
  follow?: boolean;
  timestamps?: boolean;
  since?: string;
  until?: string;
  tail?: string | number;
  details?: boolean;
  stdout?: boolean;
  stderr?: boolean;
}

export interface ParsedContainerLogEntry {
  raw: string;
  timestamp?: string;
  message: string;
}

const DEFAULT_OPTIONS: Required<
  Pick<
    ContainerLogsOptions,
    "follow" | "timestamps" | "tail" | "details" | "stdout" | "stderr"
  >
> = {
  follow: false,
  timestamps: true,
  tail: "100",
  details: false,
  stdout: true,
  stderr: true,
};

function buildLogsUrl(id: string, options?: ContainerLogsOptions) {
  const query = new URLSearchParams();
  const merged: ContainerLogsOptions = {
    ...DEFAULT_OPTIONS,
    ...options,
  };

  if (merged.follow !== undefined) {
    query.set("follow", String(merged.follow));
  }
  if (merged.timestamps !== undefined) {
    query.set("timestamps", String(merged.timestamps));
  }
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

  const path = `${BASE_URL}/${encodeURIComponent(id)}/logs`;
  const queryString = query.toString();
  return queryString ? `${path}?${queryString}` : path;
}

export async function getContainerLogs(
  id: string,
  options?: ContainerLogsOptions
): Promise<string> {
  const response = await authenticatedFetch(buildLogsUrl(id, options), {
    headers: {
      Accept: "text/plain",
    },
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Failed to fetch logs for container ${id}`);
  }

  return response.text();
}

export async function streamContainerLogs(
  id: string,
  options?: ContainerLogsOptions
): Promise<ReadableStream<Uint8Array>> {
  const response = await authenticatedFetch(buildLogsUrl(id, options), {
    headers: {
      Accept: "text/plain",
    },
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Failed to stream logs for container ${id}`);
  }

  if (!response.body) {
    throw new Error("Streaming logs are not supported in this environment.");
  }

  return response.body;
}

async function* iterateLinesFromStream(stream: ReadableStream<Uint8Array>) {
  const reader = stream.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let completed = false;

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }

      buffer += decoder.decode(value, { stream: true });

      const segments = buffer.split(/\r?\n/);
      buffer = segments.pop() ?? "";

      for (const segment of segments) {
        yield segment;
      }
    }

    buffer += decoder.decode();
    if (buffer) {
      yield buffer;
    }
    completed = true;
  } finally {
    if (!completed) {
      await reader.cancel().catch(() => {});
    }
    reader.releaseLock();
  }
}

export async function* streamContainerLogsLines(
  id: string,
  options?: ContainerLogsOptions
): AsyncGenerator<string, void, unknown> {
  const stream = await streamContainerLogs(id, options);
  for await (const line of iterateLinesFromStream(stream)) {
    yield line;
  }
}

const TIMESTAMP_REGEX =
  /^(\d{4}-\d{2}-\d{2}T[0-9:.+\-Z]+)\s+(.*)$/;

export function parseContainerLogLine(
  line: string
): ParsedContainerLogEntry {
  const match = line.match(TIMESTAMP_REGEX);
  if (match) {
    return {
      raw: line,
      timestamp: match[1],
      message: match[2] ?? "",
    };
  }

  return {
    raw: line,
    message: line,
  };
}

export async function* streamContainerLogsEntries(
  id: string,
  options?: ContainerLogsOptions
): AsyncGenerator<ParsedContainerLogEntry, void, unknown> {
  const stream = await streamContainerLogs(id, options);
  for await (const line of iterateLinesFromStream(stream)) {
    yield parseContainerLogLine(line);
  }
}

export function getContainerLogsUrl(
  id: string,
  options?: ContainerLogsOptions
): string {
  return buildLogsUrl(id, options);
}
