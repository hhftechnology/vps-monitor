import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { BotSection } from "./bot-section";
import type { BotConfig } from "../types";

vi.mock("../hooks/use-settings", () => ({
	useUpdateBot: () => ({ isPending: false, mutate: vi.fn() }),
	useTestBot: () => ({ isPending: false, mutate: vi.fn() }),
	useTestDiscordBot: () => ({ isPending: false, mutate: vi.fn() }),
}));

const baseConfig: BotConfig = {
	source: "file",
	enabled: true,
	mode: "polling",
	telegramTokenConfigured: true,
	allowedChatId: "123",
	relayPath: "/api/v1/bot/relay/command",
	relayUsesAuth: true,
	discord: {
		enabled: false,
		botToken: "",
		applicationId: "",
		guildId: "",
		allowedChannelId: "",
	},
};

describe("BotSection", () => {
	it("uses a masked token placeholder when the token is configured", () => {
		render(<BotSection config={baseConfig} />);

		expect((screen.getByLabelText("Telegram token") as HTMLInputElement).value).toBe("••••••••");
	});

	it("disables controls for mixed env-backed bot config", () => {
		render(<BotSection config={{ ...baseConfig, source: "mixed" }} />);

		expect((screen.getByLabelText("Telegram token") as HTMLInputElement).disabled).toBe(true);
		expect(screen.queryByRole("button", { name: /save changes/i })).toBeNull();
	});
});
