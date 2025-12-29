import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { ImageRemoveResult } from "../types";

const IMAGES_ENDPOINT = `${API_BASE_URL}/api/v1/images`;

export interface RemoveImageParams {
  imageId: string;
  host: string;
  force?: boolean;
}

export async function removeImage({
  imageId,
  host,
  force = false,
}: RemoveImageParams): Promise<ImageRemoveResult> {
  const params = new URLSearchParams({
    host,
    force: String(force),
  });

  const response = await authenticatedFetch(
    `${IMAGES_ENDPOINT}/${encodeURIComponent(imageId)}?${params}`,
    { method: "DELETE" }
  );

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  return (await response.json()) as ImageRemoveResult;
}
