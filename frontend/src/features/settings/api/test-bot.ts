import { authenticatedFetch } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

const ENDPOINT = `${API_BASE_URL}/api/v1/settings/test/bot`;

export async function testBot(telegramToken: string, allowedChatId: string) {
	const response = await authenticatedFetch(ENDPOINT, {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ telegramToken, allowedChatId }),
	});

	if (!response.ok) {
		const message = await response.text();
		throw new Error(message || "Failed to test bot");
	}

	return response.json() as Promise<{ success: boolean; message: string }>;
}

export async function testDiscordBot(botToken: string, allowedChannelId: string) {
	const response = await authenticatedFetch(
		`${API_BASE_URL}/api/v1/settings/test/discord-bot`,
		{
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ botToken, allowedChannelId }),
		},
	);

	if (!response.ok) {
		const message = await response.text();
		throw new Error(message || "Failed to test Discord bot");
	}

	return response.json() as Promise<{ success: boolean; message: string }>;
}
