import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

const ENDPOINT = `${API_BASE_URL}/api/v1/settings/docker-hosts`;

export async function updateDockerHosts(hosts: { name: string; host: string }[]): Promise<string> {
  const response = await authenticatedFetch(ENDPOINT, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ hosts }),
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || "Failed to update Docker hosts");
  }

  const data = (await response.json()) as { message?: string };
  return data.message ?? "Docker hosts updated";
}
