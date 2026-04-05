import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { ScanJob, ScannerType } from "../types";

const SCAN_ENDPOINT = `${API_BASE_URL}/api/v1/scan`;

export interface StartScanParams {
  imageRef: string;
  host: string;
  scanner?: ScannerType;
}

export class RescanBlockedError extends Error {
  lastScanId?: string;
  lastScanAt?: number;
  constructor(data: { message: string; last_scan_id?: string; last_scan_at?: number }) {
    super(data.message);
    this.name = "RescanBlockedError";
    this.lastScanId = data.last_scan_id;
    this.lastScanAt = data.last_scan_at;
  }
}

export async function startScan(params: StartScanParams): Promise<ScanJob> {
  const response = await authenticatedFetch(SCAN_ENDPOINT, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
  });

  if (response.status === 409) {
    const data = await response.json();
    throw new RescanBlockedError(data);
  }

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.job as ScanJob;
}
