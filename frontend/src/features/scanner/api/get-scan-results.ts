import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { ScanResult } from "../types";

const SCAN_RESULTS_ENDPOINT = `${API_BASE_URL}/api/v1/scan/results`;

export async function getScanResults(imageRef: string, host: string): Promise<ScanResult[]> {
  const encoded = encodeURIComponent(imageRef);
  const response = await authenticatedFetch(
    `${SCAN_RESULTS_ENDPOINT}?image=${encoded}&host=${encodeURIComponent(host)}`
  );

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.results as ScanResult[];
}

export async function getLatestScanResult(imageRef: string, host: string): Promise<ScanResult | null> {
  const encoded = encodeURIComponent(imageRef);
  const response = await authenticatedFetch(
    `${SCAN_RESULTS_ENDPOINT}/latest?image=${encoded}&host=${encodeURIComponent(host)}`
  );

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.result as ScanResult;
}
