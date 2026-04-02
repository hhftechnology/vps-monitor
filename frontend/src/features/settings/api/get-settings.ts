import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { SettingsResponse } from "../types";

const SETTINGS_ENDPOINT = `${API_BASE_URL}/api/v1/settings`;

export async function getSettings(): Promise<SettingsResponse> {
  const response = await authenticatedFetch(SETTINGS_ENDPOINT);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  return (await response.json()) as SettingsResponse;
}
