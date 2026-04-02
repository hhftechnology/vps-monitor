import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

const ENDPOINT = `${API_BASE_URL}/api/v1/settings/read-only`;

export async function updateReadOnly(value: boolean): Promise<string> {
  const response = await authenticatedFetch(ENDPOINT, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ value }),
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || "Failed to update read-only mode");
  }

  const data = (await response.json()) as { message?: string };
  return data.message ?? "Read-only mode updated";
}
