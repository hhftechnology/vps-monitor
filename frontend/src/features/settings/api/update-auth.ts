import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

const ENDPOINT = `${API_BASE_URL}/api/v1/settings/auth`;

export interface UpdateAuthPayload {
  enabled: boolean;
  adminUsername: string;
  newPassword?: string;
}

export async function updateAuth(payload: UpdateAuthPayload): Promise<string> {
  const response = await authenticatedFetch(ENDPOINT, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || "Failed to update auth settings");
  }

  const data = (await response.json()) as { message?: string };
  return data.message ?? "Auth settings updated";
}
