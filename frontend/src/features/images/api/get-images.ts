import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { ImageInfo } from "../types";

const IMAGES_ENDPOINT = `${API_BASE_URL}/api/v1/images`;

export interface GetImagesResponse {
  images: ImageInfo[];
  readOnly: boolean;
}

export async function getImages(): Promise<GetImagesResponse> {
  const response = await authenticatedFetch(IMAGES_ENDPOINT);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = (await response.json()) as unknown;

  if (!data || typeof data !== "object" || data === null) {
    throw new Error("Unexpected response format");
  }

  const images = (data as { images?: unknown }).images;
  const readOnly = (data as { readOnly?: boolean }).readOnly ?? false;

  if (!Array.isArray(images)) {
    throw new Error("Unexpected response format");
  }

  return {
    images: images as ImageInfo[],
    readOnly,
  };
}
