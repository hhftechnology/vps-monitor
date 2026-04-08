import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { SBOMFormat, SBOMJob, SBOMRescanBlockedResponse } from "../types";

const SBOM_ENDPOINT = `${API_BASE_URL}/api/v1/scan/sbom`;

export interface GenerateSBOMParams {
  imageRef: string;
  host: string;
  format?: SBOMFormat;
  force?: boolean;
}

export class SBOMRegenBlockedError extends Error {
  readonly data: SBOMRescanBlockedResponse;

  constructor(data: SBOMRescanBlockedResponse) {
    super(data.message);
    this.name = "SBOMRegenBlockedError";
    this.data = data;
  }
}

export async function generateSBOM(params: GenerateSBOMParams): Promise<SBOMJob> {
  const response = await authenticatedFetch(SBOM_ENDPOINT, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
  });

  if (response.status === 409) {
    const data = await response.json();
    throw new SBOMRegenBlockedError(data as SBOMRescanBlockedResponse);
  }

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.job as SBOMJob;
}

export async function getSBOMJob(id: string): Promise<SBOMJob> {
  const response = await authenticatedFetch(`${SBOM_ENDPOINT}/${id}`);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = await response.json();
  return data.job as SBOMJob;
}

export function getSBOMDownloadURL(id: string): string {
  return `${SBOM_ENDPOINT}/${id}?download=true`;
}

export async function downloadSBOM(id: string): Promise<Blob> {
  const response = await authenticatedFetch(getSBOMDownloadURL(id));

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  return response.blob();
}
