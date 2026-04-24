import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

const BASE_URL = `${API_BASE_URL}/api/v1/containers`;

type ContainerAction = "start" | "stop" | "restart" | "remove";

interface ActionResponse {
	message?: string;
	status?: string;
}

export interface ActionResult {
	message: string;
	isPending: boolean;
}

async function performContainerAction(
	id: string,
	action: ContainerAction,
	host: string,
): Promise<ActionResult> {
	const endpoint = `${BASE_URL}/${encodeURIComponent(id)}/${action}?host=${encodeURIComponent(host)}`;
	const response = await authenticatedFetch(endpoint, {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
		},
	});

	if (!response.ok) {
		const message = await response.text();
		throw new Error(message || `Failed to ${action} container`);
	}

	const isPending = response.status === 202;
	const data = (await response.json()) as ActionResponse | undefined;

	return {
		message: data?.message || "Action completed successfully",
		isPending,
	};
}

export function startContainer(id: string, host: string) {
	return performContainerAction(id, "start", host);
}

export function stopContainer(id: string, host: string) {
	return performContainerAction(id, "stop", host);
}

export function restartContainer(id: string, host: string) {
	return performContainerAction(id, "restart", host);
}

export function removeContainer(id: string, host: string) {
	return performContainerAction(id, "remove", host);
}
