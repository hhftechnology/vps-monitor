import { useEffect, useState } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
	Card,
	CardContent,
	CardDescription,
	CardHeader,
	CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";

import { useTestBot, useUpdateBot } from "../hooks/use-settings";
import type { BotConfig } from "../types";
import { EnvBadge } from "./env-badge";

interface BotSectionProps {
	config: BotConfig;
	disabled?: boolean;
}

export function BotSection({ config, disabled = false }: BotSectionProps) {
	const isEnv = config.source === "env";
	const [enabled, setEnabled] = useState(config.enabled);
	const [telegramToken, setTelegramToken] = useState(config.telegramToken);
	const [allowedChatId, setAllowedChatId] = useState(config.allowedChatId);

	const updateMutation = useUpdateBot();
	const testMutation = useTestBot();

	useEffect(() => {
		setEnabled(config.enabled);
		setTelegramToken(config.telegramToken);
		setAllowedChatId(config.allowedChatId);
	}, [config]);

	const hasChanges =
		enabled !== config.enabled ||
		telegramToken !== config.telegramToken ||
		allowedChatId !== config.allowedChatId;

	const controlsDisabled =
		disabled || isEnv || updateMutation.isPending || testMutation.isPending;

	const handleSave = () => {
		updateMutation.mutate(
			{
				enabled,
				telegramToken,
				allowedChatId,
			},
			{
				onSuccess: (message) => toast.success(message),
				onError: (error) => toast.error(error.message),
			},
		);
	};

	const handleTest = () => {
		testMutation.mutate(
			{ telegramToken, allowedChatId },
			{
				onSuccess: (result) => {
					if (result.success) {
						toast.success(result.message);
					} else {
						toast.error(result.message);
					}
				},
				onError: (error) => toast.error(error.message),
			},
		);
	};

	return (
		<Card>
			<CardHeader>
				<div className="flex items-center gap-3">
					<CardTitle>Telegram Bot</CardTitle>
					{isEnv && <EnvBadge />}
				</div>
				<CardDescription>
					Configure the Telegram bot for `/help`, `/status`, and `/critical`
					commands.
				</CardDescription>
			</CardHeader>
			<CardContent className="space-y-4">
				<div className="flex items-center gap-3">
					<Switch
						id="bot-enabled"
						checked={enabled}
						onCheckedChange={setEnabled}
						disabled={controlsDisabled}
					/>
					<Label htmlFor="bot-enabled" className="cursor-pointer">
						{enabled ? "Enabled" : "Disabled"}
					</Label>
				</div>

				<div className="grid gap-4 sm:grid-cols-2">
					<div className="space-y-1.5">
						<Label htmlFor="telegram-token">Telegram token</Label>
						<Input
							id="telegram-token"
							value={telegramToken}
							onChange={(event) => setTelegramToken(event.target.value)}
							disabled={controlsDisabled}
							type="password"
							placeholder="123456:ABC..."
						/>
					</div>
					<div className="space-y-1.5">
						<Label htmlFor="allowed-chat-id">Allowed chat ID</Label>
						<Input
							id="allowed-chat-id"
							value={allowedChatId}
							onChange={(event) => setAllowedChatId(event.target.value)}
							disabled={controlsDisabled}
							placeholder="123456789"
						/>
					</div>
				</div>

				{!isEnv && (
					<div className="flex items-center gap-2">
						<Button
							size="sm"
							onClick={handleSave}
							disabled={!hasChanges || controlsDisabled}
						>
							{updateMutation.isPending ? (
								<>
									<Spinner className="size-3" />
									Saving...
								</>
							) : (
								"Save changes"
							)}
						</Button>
						<Button
							size="sm"
							variant="outline"
							onClick={handleTest}
							disabled={controlsDisabled}
						>
							{testMutation.isPending ? "Testing..." : "Send test"}
						</Button>
					</div>
				)}
			</CardContent>
		</Card>
	);
}
