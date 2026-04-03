import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { BulkScanJob, ScanJob } from "../types";

const SCAN_JOBS_ENDPOINT = `${API_BASE_URL}/api/v1/scan/jobs`;

export interface GetScanJobsResponse {
  jobs: ScanJob[];
  bulkJobs: BulkScanJob[];
}

export async function getScanJobs(): Promise<GetScanJobsResponse> {
  const response = await authenticatedFetch(SCAN_JOBS_ENDPOINT);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  return response.json();
}

export async function getScanJob(id: string): Promise<{ job?: ScanJob; bulkJob?: BulkScanJob }> {
  const response = await authenticatedFetch(`${SCAN_JOBS_ENDPOINT}/${id}`);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  return response.json();
}

export async function cancelScanJob(id: string): Promise<void> {
  const response = await authenticatedFetch(`${SCAN_JOBS_ENDPOINT}/${id}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }
}
