import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { getSettings } from "../api/get-settings";
import { testBot, testDiscordBot } from "../api/test-bot";
import { testCoolifyHost } from "../api/test-coolify-host";
import { testDockerHost } from "../api/test-docker-host";
import { type UpdateAuthPayload, updateAuth } from "../api/update-auth";
import { type UpdateBotPayload, updateBot } from "../api/update-bot";
import { updateCoolifyHosts } from "../api/update-coolify-hosts";
import { updateDockerHosts } from "../api/update-docker-hosts";
import { updateReadOnly } from "../api/update-read-only";

const SETTINGS_KEY = ["settings"] as const;

export function useSettings() {
	return useQuery({
		queryKey: SETTINGS_KEY,
		queryFn: getSettings,
		staleTime: 30_000,
	});
}

export function useUpdateDockerHosts() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (hosts: { name: string; host: string }[]) =>
			updateDockerHosts(hosts),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: SETTINGS_KEY });
		},
	});
}

export function useUpdateCoolifyHosts() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (
			hosts: { hostName: string; apiURL: string; apiToken: string }[],
		) => updateCoolifyHosts(hosts),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: SETTINGS_KEY });
		},
	});
}

export function useUpdateReadOnly() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (value: boolean) => updateReadOnly(value),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: SETTINGS_KEY });
		},
	});
}

export function useUpdateAuth() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (payload: UpdateAuthPayload) => updateAuth(payload),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: SETTINGS_KEY });
		},
	});
}

export function useUpdateBot() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (payload: UpdateBotPayload) => updateBot(payload),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: SETTINGS_KEY });
		},
	});
}

export function useTestDockerHost() {
	return useMutation({
		mutationFn: ({ name, host }: { name: string; host: string }) =>
			testDockerHost(name, host),
	});
}

export function useTestCoolifyHost() {
	return useMutation({
		mutationFn: ({
			hostName,
			apiURL,
			apiToken,
		}: {
			hostName: string;
			apiURL: string;
			apiToken: string;
		}) => testCoolifyHost(hostName, apiURL, apiToken),
	});
}

export function useTestBot() {
	return useMutation({
		mutationFn: ({
			telegramToken,
			allowedChatId,
		}: {
			telegramToken: string;
			allowedChatId: string;
		}) => testBot(telegramToken, allowedChatId),
	});
}

export function useTestDiscordBot() {
	return useMutation({
		mutationFn: ({
			botToken,
			allowedChannelId,
		}: {
			botToken: string;
			allowedChannelId: string;
		}) => testDiscordBot(botToken, allowedChannelId),
	});
}
