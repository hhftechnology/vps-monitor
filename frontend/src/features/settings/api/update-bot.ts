import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

const ENDPOINT = `${API_BASE_URL}/api/v1/settings/bot`;

export interface UpdateBotPayload {
	enabled: boolean;
	mode: "polling" | "jwt-relay";
	telegramToken: string;
	allowedChatId: string;
}

export async function updateBot(payload: UpdateBotPayload): Promise<string> {
	const response = await authenticatedFetch(ENDPOINT, {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify(payload),
	});

	if (!response.ok) {
		const message = await response.text();
		throw new Error(message || "Failed to update bot settings");
	}

	const data = (await response.json()) as { message?: string };
	return data.message ?? "Bot settings updated";
}
