import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { ContainerInfo } from "../types";
import type { ContainerStats } from "../types/stats";

export interface ContainerHistoryStats
	extends NonNullable<ContainerInfo["historical_stats"]> {
	has_data: boolean;
	samples: ContainerStats[];
}

export async function getContainerHistory(
	id: string,
	host: string,
): Promise<ContainerHistoryStats> {
	const endpoint = `${API_BASE_URL}/api/v1/containers/${encodeURIComponent(id)}/stats/history?host=${encodeURIComponent(host)}`;
	const response = await authenticatedFetch(endpoint);

	if (!response.ok) {
		const message = await response.text();
		throw new Error(message || `Request failed with status ${response.status}`);
	}

	return response.json() as Promise<ContainerHistoryStats>;
}
