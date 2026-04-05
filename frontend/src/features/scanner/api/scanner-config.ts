import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { ScannerConfig } from "../types";

const SCANNER_CONFIG_ENDPOINT = `${API_BASE_URL}/api/v1/settings/scan`;

export async function getScannerConfig(): Promise<ScannerConfig> {
  const response = await authenticatedFetch(SCANNER_CONFIG_ENDPOINT);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.config as ScannerConfig;
}

export async function updateScannerConfig(config: ScannerConfig): Promise<ScannerConfig> {
  const response = await authenticatedFetch(SCANNER_CONFIG_ENDPOINT, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.config as ScannerConfig;
}

export async function testScanNotification(): Promise<void> {
  const response = await authenticatedFetch(`${SCANNER_CONFIG_ENDPOINT}/test-notification`, {
    method: "POST",
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }
}
