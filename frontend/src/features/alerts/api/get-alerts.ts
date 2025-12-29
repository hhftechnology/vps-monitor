import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { Alert, AlertConfig } from "../types";

const ALERTS_ENDPOINT = `${API_BASE_URL}/api/v1/alerts`;

export interface GetAlertsResponse {
  alerts: Alert[];
}

export async function getAlerts(): Promise<GetAlertsResponse> {
  const response = await authenticatedFetch(ALERTS_ENDPOINT);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = (await response.json()) as unknown;

  if (!data || typeof data !== "object" || data === null) {
    throw new Error("Unexpected response format");
  }

  const alerts = (data as { alerts?: unknown }).alerts;

  if (!Array.isArray(alerts)) {
    return { alerts: [] };
  }

  return {
    alerts: alerts as Alert[],
  };
}

export async function getAlertConfig(): Promise<AlertConfig> {
  const response = await authenticatedFetch(`${ALERTS_ENDPOINT}/config`);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  return (await response.json()) as AlertConfig;
}

export async function acknowledgeAlert(alertId: string): Promise<void> {
  const response = await authenticatedFetch(
    `${ALERTS_ENDPOINT}/${encodeURIComponent(alertId)}/acknowledge`,
    { method: "POST" }
  );

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }
}
