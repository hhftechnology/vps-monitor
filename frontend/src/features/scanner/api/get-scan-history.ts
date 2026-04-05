import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { HistoryPage, HistoryQueryParams, ScannedImage, ScanResult, AutoScanStatus } from "../types";

const HISTORY_ENDPOINT = `${API_BASE_URL}/api/v1/scan/history`;
const AUTOSCAN_ENDPOINT = `${API_BASE_URL}/api/v1/scan/autoscan/status`;

export async function getScanHistory(params: HistoryQueryParams): Promise<HistoryPage> {
  const searchParams = new URLSearchParams();

  if (params.image) searchParams.set("image", params.image);
  if (params.host) searchParams.set("host", params.host);
  if (params.min_severity) searchParams.set("min_severity", params.min_severity);
  if (params.start_date) searchParams.set("start_date", String(params.start_date));
  if (params.end_date) searchParams.set("end_date", String(params.end_date));
  if (params.page) searchParams.set("page", String(params.page));
  if (params.page_size) searchParams.set("page_size", String(params.page_size));
  if (params.sort_by) searchParams.set("sort_by", params.sort_by);
  if (params.sort_dir) searchParams.set("sort_dir", params.sort_dir);

  const response = await authenticatedFetch(
    `${HISTORY_ENDPOINT}?${searchParams.toString()}`
  );

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  return response.json();
}

export async function getScanHistoryDetail(id: string): Promise<ScanResult> {
  const response = await authenticatedFetch(`${HISTORY_ENDPOINT}/${id}`);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.result as ScanResult;
}

export async function getScannedImages(): Promise<ScannedImage[]> {
  const response = await authenticatedFetch(`${HISTORY_ENDPOINT}/images`);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.images as ScannedImage[];
}

export async function getAutoScanStatus(): Promise<AutoScanStatus> {
  const response = await authenticatedFetch(AUTOSCAN_ENDPOINT);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  return response.json();
}

export async function deleteScanHistory(id: string): Promise<void> {
  const response = await authenticatedFetch(`${HISTORY_ENDPOINT}/${id}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }
}

export async function exportScanHistory(id: string): Promise<void> {
  const response = await authenticatedFetch(`${HISTORY_ENDPOINT}/${id}/export`, {
    method: "GET",
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `scan_${id}.csv`;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  window.URL.revokeObjectURL(url);
}
