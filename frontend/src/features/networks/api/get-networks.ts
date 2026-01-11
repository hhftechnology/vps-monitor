import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { NetworkDetails, NetworkInfo } from "../types";

const NETWORKS_ENDPOINT = `${API_BASE_URL}/api/v1/networks`;

export interface GetNetworksResponse {
  networks: NetworkInfo[];
}

export async function getNetworks(): Promise<GetNetworksResponse> {
  const response = await authenticatedFetch(NETWORKS_ENDPOINT);

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = (await response.json()) as unknown;

  if (!data || typeof data !== "object" || data === null) {
    throw new Error("Unexpected response format");
  }

  const networks = (data as { networks?: unknown }).networks;

  if (!Array.isArray(networks)) {
    throw new Error("Unexpected response format");
  }

  return {
    networks: networks as NetworkInfo[],
  };
}

export async function getNetworkDetails(
  networkId: string,
  host: string
): Promise<NetworkDetails> {
  const params = new URLSearchParams({ host });
  const response = await authenticatedFetch(
    `${NETWORKS_ENDPOINT}/${encodeURIComponent(networkId)}?${params}`
  );

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Request failed with status ${response.status}`);
  }

  const data = (await response.json()) as unknown;

  if (!data || typeof data !== "object" || data === null) {
    throw new Error("Unexpected response format");
  }

  // Backend returns { network: NetworkDetails }
  const network = (data as { network?: unknown }).network;

  if (!network || typeof network !== "object") {
    throw new Error("Unexpected response format");
  }

  return network as NetworkDetails;
}
