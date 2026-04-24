import {
	ActivityIcon,
	ChevronDownIcon,
	ChevronRightIcon,
	FileTextIcon,
	PlayIcon,
	RotateCwIcon,
	SquareIcon,
	Trash2Icon,
} from "lucide-react";
import { Fragment } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import {
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger,
} from "@/components/ui/tooltip";
import type { ContainerInfo } from "../types";
import type {
	ContainerActionType,
	GroupByOption,
	GroupedContainers,
	StatsInterval,
} from "./container-utils";
import {
	formatContainerName,
	formatCreatedDate,
	formatUptime,
	getHistoricalValue,
	getStateBadgeClass,
	toTitleCase,
} from "./container-utils";

interface ContainersTableProps {
	isLoading: boolean;
	isError: boolean;
	error: unknown;
	groupBy: GroupByOption;
	filteredContainers: ContainerInfo[];
	groupedItems: GroupedContainers[] | null;
	pageItems: ContainerInfo[];
	pendingAction: { id: string; type: ContainerActionType } | null;
	isReadOnly: boolean;
	expandedGroups: string[];
	selectedIds: string[];
	statsInterval: StatsInterval;
	onToggleSelect: (id: string) => void;
	onSelectAll: () => void;
	onToggleGroup: (project: string) => void;
	onStart: (container: ContainerInfo) => void;
	onStop: (container: ContainerInfo) => void;
	onRestart: (container: ContainerInfo) => void;
	onDelete: (container: ContainerInfo) => void;
	onViewLogs: (container: ContainerInfo) => void;
	onViewStats: (container: ContainerInfo) => void;
	onRetry: () => void;
}

export function ContainersTable({
	isLoading,
	isError,
	error,
	groupBy,
	filteredContainers,
	groupedItems,
	pageItems,
	pendingAction,
	isReadOnly,
	expandedGroups,
	selectedIds,
	statsInterval,
	onToggleSelect,
	onSelectAll,
	onToggleGroup,
	onStart,
	onStop,
	onRestart,
	onDelete,
	onViewLogs,
	onViewStats,
	onRetry,
}: ContainersTableProps) {
	const isContainerActionPending = (
		action: ContainerActionType,
		containerId: string,
	) => pendingAction?.id === containerId && pendingAction.type === action;

	const isContainerBusy = (containerId: string) =>
		pendingAction?.id === containerId;

	const formatHistoricalMetric = (value: number | null) =>
		value === null ? "Collecting" : `${value.toFixed(1)}%`;

	const renderContainerRow = (container: ContainerInfo) => {
		const state = container.state.toLowerCase();
		const busy = isContainerBusy(container.id);
		const startPending = isContainerActionPending("start", container.id);
		const stopPending = isContainerActionPending("stop", container.id);
		const restartPending = isContainerActionPending("restart", container.id);
		const removePending = isContainerActionPending("remove", container.id);
		const cpuAverage = getHistoricalValue(container, statsInterval, "cpu");
		const memoryAverage = getHistoricalValue(
			container,
			statsInterval,
			"memory",
		);

		return (
			<TableRow key={container.id} className="hover:bg-muted/50">
				<TableCell className="w-10 px-4">
					<input
						type="checkbox"
						checked={selectedIds.includes(container.id)}
						onChange={() => onToggleSelect(container.id)}
						aria-label={`Select ${formatContainerName(container.names)}`}
					/>
				</TableCell>
				<TableCell className="h-16 px-4 font-medium">
					{formatContainerName(container.names)}
				</TableCell>
				<TableCell className="h-16 px-4 text-sm text-muted-foreground">
					<TooltipProvider>
						<Tooltip>
							<TooltipTrigger asChild>
								<button
									type="button"
									className="block max-w-[260px] cursor-pointer truncate"
									onClick={() => {
										navigator.clipboard?.writeText(container.image);
									}}
									title="Click to copy image name"
									aria-label={`Copy ${container.image}`}
								>
									{container.image}
								</button>
							</TooltipTrigger>
							<TooltipContent className="max-w-md break-all">
								{container.image}
							</TooltipContent>
						</Tooltip>
					</TooltipProvider>
				</TableCell>
				<TableCell className="h-16 px-4">
					<Badge className={`${getStateBadgeClass(container.state)} border-0`}>
						{toTitleCase(container.state)}
					</Badge>
				</TableCell>
				<TableCell className="h-16 px-4 text-sm text-muted-foreground">
					{formatUptime(container.created)}
				</TableCell>
				<TableCell className="h-16 px-4 text-sm text-muted-foreground">
					{formatCreatedDate(container.created)}
				</TableCell>
				<TableCell className="h-16 px-4 text-sm text-muted-foreground">
					{formatHistoricalMetric(cpuAverage)}
				</TableCell>
				<TableCell className="h-16 px-4 text-sm text-muted-foreground">
					{formatHistoricalMetric(memoryAverage)}
				</TableCell>
				<TableCell className="h-16 max-w-[300px] px-4 text-sm text-muted-foreground">
					<TooltipProvider>
						<Tooltip>
							<TooltipTrigger asChild>
								<span className="block max-w-[280px] cursor-help truncate">
									{container.command}
								</span>
							</TooltipTrigger>
							<TooltipContent className="max-w-md break-all">
								{container.command}
							</TooltipContent>
						</Tooltip>
					</TooltipProvider>
				</TableCell>
				<TableCell className="h-16 px-4">
					<TooltipProvider>
						<div className="flex items-center gap-1">
							{state === "exited" && (
								<Tooltip>
									<TooltipTrigger asChild>
										<span className="inline-block">
											<Button
												variant="outline"
												size="icon"
												className="h-8 w-8"
												onClick={() => onStart(container)}
												disabled={busy || isReadOnly}
												aria-label={`Start container ${formatContainerName(container.names)}`}
											>
												{startPending ? (
													<Spinner className="size-4" />
												) : (
													<PlayIcon className="size-4" />
												)}
											</Button>
										</span>
									</TooltipTrigger>
									<TooltipContent>
										{isReadOnly ? "Start (Read-only mode)" : "Start"}
									</TooltipContent>
								</Tooltip>
							)}
							{state === "running" && (
								<Tooltip>
									<TooltipTrigger asChild>
										<span className="inline-block">
											<Button
												variant="outline"
												size="icon"
												className="h-8 w-8"
												onClick={() => onStop(container)}
												disabled={busy || isReadOnly}
												aria-label={`Stop container ${formatContainerName(container.names)}`}
											>
												{stopPending ? (
													<Spinner className="size-4" />
												) : (
													<SquareIcon className="size-4" />
												)}
											</Button>
										</span>
									</TooltipTrigger>
									<TooltipContent>
										{isReadOnly ? "Stop (Read-only mode)" : "Stop"}
									</TooltipContent>
								</Tooltip>
							)}
							<Tooltip>
								<TooltipTrigger asChild>
									<span className="inline-block">
										<Button
											variant="outline"
											size="icon"
											className="h-8 w-8"
											onClick={() => onRestart(container)}
											disabled={busy || isReadOnly}
											aria-label={`Restart container ${formatContainerName(container.names)}`}
										>
											{restartPending ? (
												<Spinner className="size-4" />
											) : (
												<RotateCwIcon className="size-4" />
											)}
										</Button>
									</span>
								</TooltipTrigger>
								<TooltipContent>
									{isReadOnly ? "Restart (Read-only mode)" : "Restart"}
								</TooltipContent>
							</Tooltip>
							<Tooltip>
								<TooltipTrigger asChild>
									<span className="inline-block">
										<Button
											variant="outline"
											size="icon"
											className="h-8 w-8 text-destructive hover:bg-destructive hover:text-white"
											onClick={() => onDelete(container)}
											disabled={busy || isReadOnly}
											aria-label={`Delete container ${formatContainerName(container.names)}`}
										>
											{removePending ? (
												<Spinner className="size-4" />
											) : (
												<Trash2Icon className="size-4" />
											)}
										</Button>
									</span>
								</TooltipTrigger>
								<TooltipContent>
									{isReadOnly ? "Delete (Read-only mode)" : "Delete"}
								</TooltipContent>
							</Tooltip>
							<Tooltip>
								<TooltipTrigger asChild>
									<Button
										variant="outline"
										size="icon"
										className="h-8 w-8"
										onClick={() => onViewLogs(container)}
										disabled={busy}
										aria-label={`View logs for container ${formatContainerName(container.names)}`}
									>
										<FileTextIcon className="size-4" />
									</Button>
								</TooltipTrigger>
								<TooltipContent>View Logs</TooltipContent>
							</Tooltip>
							<Tooltip>
								<TooltipTrigger asChild>
									<Button
										variant="outline"
										size="icon"
										className="h-8 w-8"
										onClick={() => onViewStats(container)}
										disabled={busy}
										aria-label={`View stats for container ${formatContainerName(container.names)}`}
									>
										<ActivityIcon className="size-4" />
									</Button>
								</TooltipTrigger>
								<TooltipContent>View Stats</TooltipContent>
							</Tooltip>
						</div>
					</TooltipProvider>
				</TableCell>
			</TableRow>
		);
	};

	return (
		<div className="overflow-x-auto rounded-lg border bg-card">
			<Table className="min-w-[1180px]">
				<TableHeader>
					<TableRow className="hover:bg-transparent border-b">
						<TableHead className="h-12 w-10 px-4">
							<input
								type="checkbox"
								checked={
									pageItems.length > 0 &&
									selectedIds.length === pageItems.length
								}
								onChange={onSelectAll}
								aria-label="Select all containers on this page"
							/>
						</TableHead>
						<TableHead className="h-12 px-4 font-medium">Name</TableHead>
						<TableHead className="h-12 px-4 font-medium">Image</TableHead>
						<TableHead className="h-12 px-4 font-medium w-[120px]">
							State
						</TableHead>
						<TableHead className="h-12 px-4 font-medium">Uptime</TableHead>
						<TableHead className="h-12 px-4 font-medium">Created</TableHead>
						<TableHead className="h-12 px-4 font-medium">
							CPU {statsInterval}
						</TableHead>
						<TableHead className="h-12 px-4 font-medium">
							RAM {statsInterval}
						</TableHead>
						<TableHead className="h-12 px-4 font-medium">Command</TableHead>
						<TableHead className="h-12 px-4 font-medium w-[160px]">
							Actions
						</TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{isLoading ? (
						<TableRow>
							<TableCell colSpan={10} className="h-32">
								<div className="flex items-center justify-center text-sm text-muted-foreground">
									<Spinner className="mr-2" />
									Loading containers…
								</div>
							</TableCell>
						</TableRow>
					) : isError ? (
						<TableRow>
							<TableCell colSpan={10} className="h-32">
								<div className="flex flex-col items-center gap-3 text-center">
									<p className="text-sm text-muted-foreground">
										{(error as Error)?.message || "Unable to load containers."}
									</p>
									<Button size="sm" variant="outline" onClick={onRetry}>
										Try again
									</Button>
								</div>
							</TableCell>
						</TableRow>
					) : filteredContainers.length === 0 ? (
						<TableRow>
							<TableCell colSpan={10} className="h-32">
								<div className="text-center text-sm text-muted-foreground">
									No containers found.
								</div>
							</TableCell>
						</TableRow>
					) : groupBy === "compose" && groupedItems ? (
						groupedItems.map((group) => (
							<Fragment key={group.project}>
								<TableRow className="bg-muted/30 hover:bg-muted/30">
									<TableCell
										colSpan={10}
										className="h-10 px-4 text-xs font-medium text-muted-foreground"
									>
										<button
											type="button"
											className="inline-flex max-w-full items-center gap-2 truncate"
											onClick={() => onToggleGroup(group.project)}
										>
											{expandedGroups.includes(group.project) ? (
												<ChevronDownIcon className="size-4" />
											) : (
												<ChevronRightIcon className="size-4" />
											)}
											<span className="truncate">
												{group.project} · {group.items.length}{" "}
												{group.items.length === 1 ? "container" : "containers"}
											</span>
										</button>
									</TableCell>
								</TableRow>
								{expandedGroups.includes(group.project)
									? group.items.map(renderContainerRow)
									: null}
							</Fragment>
						))
					) : (
						pageItems.map(renderContainerRow)
					)}
				</TableBody>
			</Table>
		</div>
	);
}
