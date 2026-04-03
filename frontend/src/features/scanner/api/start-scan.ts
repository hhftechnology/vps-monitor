import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { ScanJob, ScannerType } from "../types";

const SCAN_ENDPOINT = `${API_BASE_URL}/api/v1/scan`;

export interface StartScanParams {
  imageRef: string;
  host: string;
  scanner?: ScannerType;
}

export async function startScan(params: StartScanParams): Promise<ScanJob> {
  const response = await authenticatedFetch(SCAN_ENDPOINT, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.job as ScanJob;
}
