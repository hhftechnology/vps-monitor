export type ConfigSource = "file" | "env" | "default" | "mixed";

export interface DockerHost {
	name: string;
	host: string;
	source: ConfigSource;
}

export interface CoolifyHost {
	hostName: string;
	apiURL: string;
	apiToken: string;
	source: ConfigSource;
}

export interface DockerHostsConfig {
	source: ConfigSource;
	hosts: DockerHost[];
}

export interface CoolifyHostsConfig {
	source: ConfigSource;
	hosts: CoolifyHost[];
}

export interface ReadOnlyConfig {
	source: ConfigSource;
	value: boolean;
}

export interface AuthConfig {
	source: ConfigSource;
	enabled: boolean;
	adminUsername?: string;
	passwordConfigured: boolean;
}

export interface BotConfig {
	source: ConfigSource;
	enabled: boolean;
	telegramToken: string;
	allowedChatId: string;
}

export interface SettingsResponse {
	dockerHosts: DockerHostsConfig;
	coolifyHosts: CoolifyHostsConfig;
	readOnly: ReadOnlyConfig;
	auth: AuthConfig;
	bot: BotConfig;
}

export interface TestConnectionResult {
	success: boolean;
	message: string;
	dockerVersion?: string;
}
