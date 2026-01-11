import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

interface EnvVariablesResponse {
  env: Record<string, string>;
}

export async function getContainerEnvVariables(
  id: string,
  host: string
): Promise<Record<string, string>> {
  const response = await authenticatedFetch(
    `${API_BASE_URL}/api/v1/containers/${encodeURIComponent(id)}/env?host=${encodeURIComponent(host)}`
  );

  if (!response.ok) {
    throw new Error("Failed to fetch container environment variables");
  }

  const data: EnvVariablesResponse = await response.json();
  return data.env || {};
}
