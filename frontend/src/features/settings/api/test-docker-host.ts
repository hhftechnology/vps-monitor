import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { TestConnectionResult } from "../types";

const ENDPOINT = `${API_BASE_URL}/api/v1/settings/test/docker-host`;

export async function testDockerHost(
  name: string,
  host: string,
): Promise<TestConnectionResult> {
  const response = await authenticatedFetch(ENDPOINT, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, host }),
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || "Failed to test Docker host");
  }

  return (await response.json()) as TestConnectionResult;
}
