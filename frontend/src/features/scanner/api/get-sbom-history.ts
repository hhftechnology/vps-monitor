import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type {
  SBOMHistoryPage,
  SBOMHistoryQueryParams,
  SBOMResult,
  SBOMedImage,
} from "../types";

const SBOM_HISTORY_ENDPOINT = `${API_BASE_URL}/api/v1/scan/sbom/history`;

export async function getSBOMHistory(params: SBOMHistoryQueryParams): Promise<SBOMHistoryPage> {
  const searchParams = new URLSearchParams();

  if (params.image) searchParams.set("image", params.image);
  if (params.host) searchParams.set("host", params.host);
  if (params.format) searchParams.set("format", params.format);
  if (params.start_date) searchParams.set("start_date", String(params.start_date));
  if (params.end_date) searchParams.set("end_date", String(params.end_date));
  if (params.page) searchParams.set("page", String(params.page));
  if (params.page_size) searchParams.set("page_size", String(params.page_size));
  if (params.sort_by) searchParams.set("sort_by", params.sort_by);
  if (params.sort_dir) searchParams.set("sort_dir", params.sort_dir);

  const response = await authenticatedFetch(
    `${SBOM_HISTORY_ENDPOINT}?${searchParams.toString()}`
  );

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  return response.json();
}

export async function getSBOMHistoryDetail(id: string): Promise<SBOMResult> {
  const response = await authenticatedFetch(`${SBOM_HISTORY_ENDPOINT}/${id}`);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.result as SBOMResult;
}

export async function getSBOMedImages(): Promise<SBOMedImage[]> {
  const response = await authenticatedFetch(`${SBOM_HISTORY_ENDPOINT}/images`);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.images as SBOMedImage[];
}

export async function deleteSBOMHistory(id: string): Promise<void> {
  const response = await authenticatedFetch(`${SBOM_HISTORY_ENDPOINT}/${id}`, {
    method: "DELETE",
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }
}

export async function downloadSBOMHistoryFile(id: string): Promise<Blob> {
  const response = await authenticatedFetch(`${SBOM_HISTORY_ENDPOINT}/${id}/download`);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  return response.blob();
}
