import {
	ActivityIcon,
	CpuIcon,
	HardDriveIcon,
	MemoryStickIcon,
	NetworkIcon,
	PlayIcon,
	SettingsIcon,
	SquareIcon,
	TerminalIcon,
} from "lucide-react";
import {
	lazy,
	Suspense,
	useCallback,
	useEffect,
	useMemo,
	useState,
} from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
	Sheet,
	SheetContent,
	SheetDescription,
	SheetHeader,
	SheetTitle,
} from "@/components/ui/sheet";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "@/components/ui/tooltip";

import { useContainerHistory } from "../hooks/use-container-history";
import { useContainerStats } from "../hooks/use-container-stats";
import type { ContainerInfo } from "../types";
import { ContainerStatsCharts } from "./container-stats-charts";
import { EnvironmentVariables } from "./environment-variables";

// Lazy load terminal to reduce initial bundle size
const Terminal = lazy(() =>
	import("./terminal").then((module) => ({ default: module.Terminal })),
);

interface ContainerDetailsSheetProps {
	container: ContainerInfo | null;
	host: string;
	isOpen: boolean;
	onOpenChange: (open: boolean) => void;
	isReadOnly?: boolean;
}

function formatBytes(bytes: number): string {
	if (bytes === 0) return "0 B";
	const k = 1024;
	const sizes = ["B", "KB", "MB", "GB", "TB"];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return `${(bytes / k ** i).toFixed(1)} ${sizes[i]}`;
}

function formatPercent(value: number): string {
	return `${value.toFixed(1)}%`;
}

export function ContainerDetailsSheet({
	container,
	host,
	isOpen,
	onOpenChange,
	isReadOnly = false,
}: ContainerDetailsSheetProps) {
	const [activeTab, setActiveTab] = useState("stats");
	const [containerId, setContainerId] = useState(container?.id ?? "");

	useEffect(() => {
		setContainerId(container?.id ?? "");
	}, [container?.id]);

	const effectiveContainerId = containerId || container?.id || "";

	const {
		stats,
		history,
		isConnected,
		error,
		connect,
		disconnect,
		clearHistory,
	} = useContainerStats({
		containerId: effectiveContainerId,
		host,
		enabled: isOpen && activeTab === "stats",
	});
	const { data: persistedHistory } = useContainerHistory(
		effectiveContainerId,
		host,
		isOpen && activeTab === "stats",
	);

	const handleContainerIdChange = useCallback((newId: string) => {
		setContainerId(newId);
	}, []);

	const handleToggleStats = useCallback(() => {
		if (isConnected) {
			disconnect();
		} else {
			clearHistory();
			connect();
		}
	}, [isConnected, disconnect, clearHistory, connect]);

	const statsCards = useMemo(() => {
		if (!stats) return null;

		return [
			{
				label: "CPU",
				value: formatPercent(stats.cpu_percent),
				icon: CpuIcon,
				color: stats.cpu_percent > 80 ? "text-red-500" : "text-primary",
			},
			{
				label: "Memory",
				value: `${formatBytes(stats.memory_usage)} / ${formatBytes(stats.memory_limit)}`,
				subValue: formatPercent(stats.memory_percent),
				icon: MemoryStickIcon,
				color: stats.memory_percent > 80 ? "text-red-500" : "text-primary",
			},
			{
				label: "Network I/O",
				value: `${formatBytes(stats.network_rx)} / ${formatBytes(stats.network_tx)}`,
				subLabel: "RX / TX",
				icon: NetworkIcon,
				color: "text-primary",
			},
			{
				label: "Block I/O",
				value: `${formatBytes(stats.block_read)} / ${formatBytes(stats.block_write)}`,
				subLabel: "Read / Write",
				icon: HardDriveIcon,
				color: "text-primary",
			},
			{
				label: "PIDs",
				value: stats.pids.toString(),
				icon: ActivityIcon,
				color: "text-primary",
			},
		];
	}, [stats]);

	const chartHistory = useMemo(() => {
		const persistedSamples = persistedHistory?.samples ?? [];
		if (persistedSamples.length === 0) {
			return history;
		}

		const merged = new Map<number, (typeof persistedSamples)[number]>();
		for (const sample of persistedSamples) {
			merged.set(sample.timestamp, sample);
		}
		for (const sample of history) {
			merged.set(sample.timestamp, sample);
		}

		return Array.from(merged.values())
			.sort((a, b) => a.timestamp - b.timestamp)
			.slice(-60);
	}, [history, persistedHistory?.samples]);

	if (!container) return null;

	const containerName =
		container.names?.[0]?.replace(/^\//, "") || container.id.slice(0, 12);
	const isRunning = container.state.toLowerCase() === "running";
	const historyStats = container.historical_stats;

	return (
		<Sheet open={isOpen} onOpenChange={onOpenChange}>
			<SheetContent className="w-full overflow-y-auto p-0 sm:max-w-4xl">
				<SheetHeader className="px-6 pt-6">
					<SheetTitle className="flex items-center gap-2">
						{containerName}
						<Badge
							variant={isRunning ? "default" : "secondary"}
							className="text-xs"
						>
							{container.state}
						</Badge>
					</SheetTitle>
					<SheetDescription className="text-xs font-mono truncate">
						{container.image}
					</SheetDescription>
				</SheetHeader>

				<Tabs
					value={activeTab}
					onValueChange={setActiveTab}
					className="flex flex-1 flex-col px-6 pb-6"
				>
					<TabsList className="grid w-full grid-cols-3">
						<TabsTrigger value="stats" className="flex items-center gap-2">
							<ActivityIcon className="size-4" />
							Stats
						</TabsTrigger>
						<TabsTrigger
							value="terminal"
							className="flex items-center gap-2"
							disabled={!isRunning}
						>
							<TerminalIcon className="size-4" />
							Terminal
						</TabsTrigger>
						<TabsTrigger value="env" className="flex items-center gap-2">
							<SettingsIcon className="size-4" />
							Env Vars
						</TabsTrigger>
					</TabsList>

					<TabsContent value="stats" className="mt-4 flex flex-col gap-6">
						{historyStats && (
							<div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
								<Card>
									<CardContent className="p-3 text-sm">
										1h CPU
										<div className="font-semibold">
											{historyStats.cpu_1h != null ? `${historyStats.cpu_1h.toFixed(1)}%` : "N/A"}
										</div>
									</CardContent>
								</Card>
								<Card>
									<CardContent className="p-3 text-sm">
										1h RAM
										<div className="font-semibold">
											{historyStats.memory_1h != null ? `${historyStats.memory_1h.toFixed(1)}%` : "N/A"}
										</div>
									</CardContent>
								</Card>
								<Card>
									<CardContent className="p-3 text-sm">
										12h CPU
										<div className="font-semibold">
											{historyStats.cpu_12h != null ? `${historyStats.cpu_12h.toFixed(1)}%` : "N/A"}
										</div>
									</CardContent>
								</Card>
								<Card>
									<CardContent className="p-3 text-sm">
										12h RAM
										<div className="font-semibold">
											{historyStats.memory_12h != null ? `${historyStats.memory_12h.toFixed(1)}%` : "N/A"}
										</div>
									</CardContent>
								</Card>
							</div>
						)}
						{/* Stats Controls */}
						<div className="flex flex-wrap items-center justify-between gap-4">
							<div className="flex items-center gap-2">
								<Badge variant={isConnected ? "default" : "secondary"}>
									{isConnected ? "Live" : "Disconnected"}
								</Badge>
								{history.length > 0 && (
									<span className="text-xs text-muted-foreground">
										{history.length} data points
									</span>
								)}
							</div>
							<Tooltip>
								<TooltipTrigger asChild>
									<Button
										variant={isConnected ? "default" : "outline"}
										size="sm"
										onClick={handleToggleStats}
										disabled={!isRunning}
									>
										{isConnected ? (
											<>
												<SquareIcon className="mr-2 size-4" />
												Stop
											</>
										) : (
											<>
												<PlayIcon className="mr-2 size-4" />
												Start
											</>
										)}
									</Button>
								</TooltipTrigger>
								<TooltipContent>
									{!isRunning
										? "Container must be running"
										: isConnected
											? "Stop streaming stats"
											: "Start streaming stats"}
								</TooltipContent>
							</Tooltip>
						</div>

						{error && <p className="text-sm text-destructive">{error}</p>}

						{!isRunning && (
							<Card>
								<CardContent className="py-8 text-center text-muted-foreground text-sm">
									Container is not running. Start the container to view stats.
								</CardContent>
							</Card>
						)}

						{isRunning && isConnected && !stats && (
							<div className="flex items-center justify-center py-8 text-muted-foreground">
								<Spinner className="mr-2 size-4" />
								Connecting...
							</div>
						)}

						{isRunning && !isConnected && !stats && (
							<Card>
								<CardContent className="py-8 text-center text-muted-foreground text-sm">
									Click "Start" to stream real-time container stats
								</CardContent>
							</Card>
						)}

						{/* Live Stats Cards */}
						{statsCards && (
							<div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-5">
								{statsCards.map((card) => {
									const Icon = card.icon;
									return (
										<Card key={card.label} className="overflow-hidden">
											<CardContent className="p-3">
												<div className="flex flex-col items-center gap-2 text-center">
													<div
														className={`p-2 rounded-lg bg-muted ${card.color}`}
													>
														<Icon className="size-5" />
													</div>
													<div className="min-w-0">
														<p className="text-xs text-muted-foreground font-medium">
															{card.label}
														</p>
														<p className="break-words text-sm font-semibold leading-tight">
															{card.value}
														</p>
														{card.subValue && (
															<p className="text-xs text-muted-foreground">
																{card.subValue}
															</p>
														)}
														{card.subLabel && (
															<p className="text-[10px] text-muted-foreground">
																{card.subLabel}
															</p>
														)}
													</div>
												</div>
											</CardContent>
										</Card>
									);
								})}
							</div>
						)}

						{/* Stats Charts */}
						<ContainerStatsCharts history={chartHistory} />
					</TabsContent>

					<TabsContent value="terminal" className="min-h-[400px] mt-4">
						{isRunning ? (
							<Suspense
								fallback={
									<div className="flex items-center justify-center h-[400px] text-muted-foreground">
										<Spinner className="mr-2 size-4" />
										Loading terminal...
									</div>
								}
							>
								<Terminal containerId={effectiveContainerId} host={host} />
							</Suspense>
						) : (
							<Card>
								<CardContent className="py-8 text-center text-muted-foreground text-sm">
									Container is not running. Start the container to access
									terminal.
								</CardContent>
							</Card>
						)}
					</TabsContent>

					<TabsContent value="env" className="mt-4">
						<EnvironmentVariables
							containerId={effectiveContainerId}
							containerHost={host}
							isReadOnly={isReadOnly}
							onContainerIdChange={handleContainerIdChange}
						/>
					</TabsContent>
				</Tabs>
			</SheetContent>
		</Sheet>
	);
}
