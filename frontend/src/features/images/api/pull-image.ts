import { getAuthToken } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { ImagePullProgress } from "../types";

const IMAGES_ENDPOINT = `${API_BASE_URL}/api/v1/images`;

export interface PullImageParams {
  imageName: string;
  host: string;
}

export async function* pullImage(
  { imageName, host }: PullImageParams,
  signal?: AbortSignal
): AsyncGenerator<ImagePullProgress> {
  const params = new URLSearchParams({
    host,
    image: imageName,
  });

  const token = getAuthToken();
  const headers: HeadersInit = {};
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${IMAGES_ENDPOINT}/pull?${params}`, {
    method: "POST",
    headers,
    signal,
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const reader = response.body?.getReader();
  if (!reader) {
    throw new Error("Response body is not readable");
  }

  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (line.trim()) {
          try {
            const progress = JSON.parse(line) as ImagePullProgress;
            yield progress;
          } catch {
            // Skip non-JSON lines
          }
        }
      }
    }

    // Process remaining buffer
    if (buffer.trim()) {
      try {
        const progress = JSON.parse(buffer) as ImagePullProgress;
        yield progress;
      } catch {
        // Skip non-JSON content
      }
    }
  } finally {
    reader.releaseLock();
  }
}
