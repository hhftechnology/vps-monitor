import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { BulkScanJob, ScannerType } from "../types";

const BULK_SCAN_ENDPOINT = `${API_BASE_URL}/api/v1/scan/bulk`;

export interface StartBulkScanParams {
  scanner?: ScannerType;
  hosts?: string[];
}

export async function startBulkScan(params: StartBulkScanParams): Promise<BulkScanJob> {
  const response = await authenticatedFetch(BULK_SCAN_ENDPOINT, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.job as BulkScanJob;
}
