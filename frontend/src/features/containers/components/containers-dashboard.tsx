import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";
import type { DateRange } from "react-day-picker";
import { toast } from "sonner";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Spinner } from "@/components/ui/spinner";
import {
	removeContainer,
	restartContainer,
	startContainer,
	stopContainer,
} from "../api/container-actions";
import type { GetContainersResponse } from "../api/get-containers";
import { useContainersDashboardUrlState } from "../hooks/use-containers-dashboard-url-state";
import { useContainersQuery } from "../hooks/use-containers-query";
import { useSystemStats } from "../hooks/use-system-stats";
import type { ContainerInfo } from "../types";
import { ContainerDetailsSheet } from "./container-details-sheet";
import type {
	ContainerActionType,
	GroupByOption,
	SortColumn,
	SortDirection,
	StatsInterval,
} from "./container-utils";
import {
	formatContainerName,
	getHistoricalValue,
	getInitialStateCounts,
	groupByCompose,
} from "./container-utils";
import { ContainersLogsSheet } from "./containers-logs-sheet";
import { ContainersPagination } from "./containers-pagination";
import { ContainersStateSummary } from "./containers-state-summary";
import { ContainersSummaryCards } from "./containers-summary-cards";
import { ContainersTable } from "./containers-table";
import { ContainersToolbar } from "./containers-toolbar";

export function ContainersDashboard() {
	const queryClient = useQueryClient();
	const { data, error, isError, isFetching, isLoading, refetch } =
		useContainersQuery();
	const { data: systemStats } = useSystemStats();

	const containers = data?.containers ?? [];
	const isReadOnly = data?.readOnly ?? false;
	const hosts = data?.hosts ?? [];
	const hostErrors = data?.hostErrors ?? [];

	const hostInfo = useMemo(
		() => ({
			hostname: systemStats?.hostInfo.hostname ?? "Loading...",
			os: systemStats?.hostInfo.platform ?? "Unknown",
			kernel: systemStats?.hostInfo.kernelVersion ?? "Unknown",
		}),
		[systemStats],
	);

	const systemUsage = useMemo(
		() => ({
			cpu: Math.round(systemStats?.usage.cpuPercent ?? 0),
			memory: Math.round(systemStats?.usage.memoryPercent ?? 0),
			disk: Math.round(systemStats?.usage.diskPercent ?? 0),
		}),
		[systemStats],
	);

	const {
		searchTerm,
		setSearchTerm,
		stateFilter,
		setStateFilter,
		hostFilter,
		setHostFilter,
		sortDirection,
		setSortDirection,
		sortBy,
		setSortBy,
		groupBy,
		setGroupBy,
		statsInterval,
		setStatsInterval,
		dateRange,
		setDateRange,
		clearDateRange,
		pageSize,
		setPageSize,
		page,
		setPage,
		expandedGroups,
		setExpandedGroups,
	} = useContainersDashboardUrlState();
	const [selectedContainer, setSelectedContainer] =
		useState<ContainerInfo | null>(null);
	const [isLogsSheetOpen, setIsLogsSheetOpen] = useState(false);
	const [isDetailsSheetOpen, setIsDetailsSheetOpen] = useState(false);
	const [detailsContainer, setDetailsContainer] =
		useState<ContainerInfo | null>(null);
	const [selectedContainerIds, setSelectedContainerIds] = useState<string[]>(
		[],
	);
	const [pendingAction, setPendingAction] = useState<{
		id: string;
		type: ContainerActionType;
	} | null>(null);
	const [confirmAction, setConfirmAction] = useState<{
		type: Extract<ContainerActionType, "stop" | "remove">;
		container: ContainerInfo;
	} | null>(null);

	// Helper function to check if a container matches filters
	const matchesFilters = useMemo(() => {
		const normalizedSearch = searchTerm.trim().toLowerCase();

		return (
			container: ContainerInfo,
			options: { includeStateFilter?: boolean } = {},
		) => {
			const matchesSearch =
				!normalizedSearch ||
				container.id.toLowerCase().startsWith(normalizedSearch) ||
				container.image.toLowerCase().includes(normalizedSearch) ||
				container.names.some((name) =>
					name.toLowerCase().includes(normalizedSearch),
				);

			const matchesHost = hostFilter === "all" || container.host === hostFilter;

			const containerDate = new Date(container.created * 1000);
			const matchesDateRange =
				!dateRange ||
				(dateRange.from &&
					dateRange.to &&
					containerDate >= dateRange.from &&
					containerDate <= dateRange.to) ||
				(dateRange.from && !dateRange.to && containerDate >= dateRange.from) ||
				(!dateRange.from && dateRange.to && containerDate <= dateRange.to);

			const matchesState = options.includeStateFilter
				? stateFilter === "all" || container.state.toLowerCase() === stateFilter
				: true;

			return matchesSearch && matchesHost && matchesDateRange && matchesState;
		};
	}, [searchTerm, hostFilter, dateRange, stateFilter]);

	const availableStates = useMemo(() => {
		const unique = new Set<string>();
		containers.forEach((container) => {
			if (container.state) {
				unique.add(container.state.toLowerCase());
			}
		});
		return Array.from(unique).sort();
	}, [containers]);

	const getSortValue = useCallback(
		(container: ContainerInfo, column: SortColumn): number | string => {
			switch (column) {
				case "name":
					return formatContainerName(container.names).toLowerCase();
				case "state":
					return container.state.toLowerCase();
				case "uptime":
				case "created":
					return container.created;
				case "cpu":
					return getHistoricalValue(container, statsInterval, "cpu") ?? -1;
				case "ram":
					return getHistoricalValue(container, statsInterval, "memory") ?? -1;
				default:
					return container.created;
			}
		},
		[statsInterval],
	);

	const filteredContainers = useMemo(() => {
		const filtered = containers.filter((container) =>
			matchesFilters(container, { includeStateFilter: true }),
		);

		return filtered.sort((a, b) => {
			const aValue = getSortValue(a, sortBy);
			const bValue = getSortValue(b, sortBy);

			if (typeof aValue === "string" && typeof bValue === "string") {
				return sortDirection === "desc"
					? bValue.localeCompare(aValue)
					: aValue.localeCompare(bValue);
			}

			return sortDirection === "desc"
				? (bValue as number) - (aValue as number)
				: (aValue as number) - (bValue as number);
		});
	}, [containers, getSortValue, matchesFilters, sortBy, sortDirection]);

	const totalPages =
		filteredContainers.length === 0
			? 1
			: Math.ceil(filteredContainers.length / pageSize);

	useEffect(() => {
		if (page > totalPages) {
			setPage(totalPages);
		}
	}, [page, totalPages, setPage]);

	const startIndex = filteredContainers.length ? (page - 1) * pageSize + 1 : 0;
	const endIndex = filteredContainers.length
		? Math.min(page * pageSize, filteredContainers.length)
		: 0;

	const pageItems = useMemo(() => {
		const offset = (page - 1) * pageSize;
		return filteredContainers.slice(offset, offset + pageSize);
	}, [filteredContainers, page, pageSize]);

	const groupedItems = useMemo(() => {
		if (groupBy !== "compose") {
			return null;
		}
		return groupByCompose(pageItems, sortDirection);
	}, [pageItems, groupBy, sortDirection]);

	const stateCounts = useMemo(() => {
		const counts = getInitialStateCounts();

		// Filter by host, search, and date - but NOT by state filter
		// This way state counts reflect the current host selection
		containers.forEach((container) => {
			if (matchesFilters(container, { includeStateFilter: false })) {
				const state = container.state.toLowerCase();
				if (state === "running") counts.running++;
				else if (state === "exited") counts.exited++;
				else if (state === "paused") counts.paused++;
				else if (state === "restarting") counts.restarting++;
				else if (state === "dead") counts.dead++;
				else counts.other++;
			}
		});

		return counts;
	}, [containers, matchesFilters]);

	const executeAction = async (
		actionType: ContainerActionType,
		container: ContainerInfo,
	) => {
		setPendingAction({ id: container.id, type: actionType });
		try {
			let result: Awaited<ReturnType<typeof startContainer>> | null = null;
			switch (actionType) {
				case "start":
					result = await startContainer(container.id, container.host);
					break;
				case "stop":
					result = await stopContainer(container.id, container.host);
					break;
				case "restart":
					result = await restartContainer(container.id, container.host);
					break;
				case "remove":
					result = await removeContainer(container.id, container.host);
					break;
				default:
					return;
			}
			if (result?.message) {
				if (result.isPending) {
					toast.info(result.message);
				} else {
					toast.success(result.message);
				}
			}
			await refetch();
		} catch (error) {
			if (error instanceof Error) {
				toast.error(error.message);
			} else {
				toast.error("Unexpected error while performing container action.");
			}
		} finally {
			setPendingAction(null);
		}
	};

	const handleConfirmAction = async () => {
		if (!confirmAction) return;
		const { type, container } = confirmAction;
		await executeAction(type, container);
		setConfirmAction(null);
	};

	const handleConfirmDialogOpenChange = (open: boolean) => {
		if (!open) {
			setConfirmAction(null);
		}
	};

	const handleSearchChange = (value: string) => {
		setSearchTerm(value);
	};

	const handleStateFilterChange = (value: string) => {
		setStateFilter(value);
	};

	const handleHostFilterChange = (value: string) => {
		setHostFilter(value);
	};

	const handleSortDirectionChange = (direction: SortDirection) => {
		setSortDirection(direction);
	};

	const handleSortByChange = (value: SortColumn) => {
		setSortBy(value);
	};

	const handleGroupByChange = (value: GroupByOption) => {
		setGroupBy(value);
	};

	const handleStatsIntervalChange = (value: StatsInterval) => {
		setStatsInterval(value);
	};

	const handleDateRangeChange = (range: DateRange | undefined) => {
		setDateRange(range);
	};

	const handleDateRangeClear = () => {
		clearDateRange();
	};

	const handlePageSizeChange = (size: number) => {
		setPageSize(size);
	};

	const handlePageChange = (nextPage: number) => {
		setPage(nextPage);
	};

	const handleViewLogs = (container: ContainerInfo) => {
		setSelectedContainer(container);
		setIsLogsSheetOpen(true);
	};

	const handleLogsSheetOpenChange = (open: boolean) => {
		setIsLogsSheetOpen(open);
		if (!open) {
			setSelectedContainer(null);
		}
	};

	const handleViewStats = (container: ContainerInfo) => {
		setDetailsContainer(container);
		setIsDetailsSheetOpen(true);
	};

	const handleDetailsSheetOpenChange = (open: boolean) => {
		setIsDetailsSheetOpen(open);
		if (!open) {
			setDetailsContainer(null);
		}
	};

	const handleContainerRecreated = async (newContainerId: string) => {
		await queryClient.refetchQueries({
			queryKey: ["containers"],
			exact: false,
		});

		const updatedData = queryClient.getQueryData<GetContainersResponse>([
			"containers",
		]);
		const newContainer = updatedData?.containers?.find(
			(c) => c.id === newContainerId,
		);

		if (newContainer) {
			setSelectedContainer(newContainer);
		}
	};

	const handleStartContainer = (container: ContainerInfo) => {
		void executeAction("start", container);
	};

	const handleStopContainer = (container: ContainerInfo) => {
		setConfirmAction({ type: "stop", container });
	};

	const handleRestartContainer = (container: ContainerInfo) => {
		void executeAction("restart", container);
	};

	const handleDeleteContainer = (container: ContainerInfo) => {
		setConfirmAction({ type: "remove", container });
	};

	const handleToggleSelect = (id: string) => {
		setSelectedContainerIds((current) =>
			current.includes(id)
				? current.filter((value) => value !== id)
				: [...current, id],
		);
	};

	const handleSelectAll = () => {
		if (selectedContainerIds.length === pageItems.length) {
			setSelectedContainerIds([]);
			return;
		}

		setSelectedContainerIds(pageItems.map((container) => container.id));
	};

	const handleToggleGroup = (project: string) => {
		setExpandedGroups(
			expandedGroups.includes(project)
				? expandedGroups.filter((value) => value !== project)
				: [...expandedGroups, project],
		);
	};

	const handleBatchAction = async (action: ContainerActionType) => {
		const selectedContainers = containers.filter((container) =>
			selectedContainerIds.includes(container.id),
		);

		for (const container of selectedContainers) {
			await executeAction(action, container);
		}

		setSelectedContainerIds([]);
	};

	useEffect(() => {
		setSelectedContainerIds((current) =>
			current.filter((id) =>
				pageItems.some((container) => container.id === id),
			),
		);
	}, [pageItems]);

	const confirmActionTitle =
		confirmAction?.type === "stop"
			? "Stop container?"
			: confirmAction?.type === "remove"
				? "Remove container?"
				: "";

	const confirmActionDescription =
		confirmAction?.type === "stop"
			? "Stopping a container will terminate its running processes."
			: confirmAction?.type === "remove"
				? "Removing a container will permanently delete it and its resources. This action cannot be undone."
				: "";

	const confirmActionButtonLabel = confirmAction
		? confirmAction.type === "stop"
			? "Stop Container"
			: "Remove Container"
		: "Confirm";

	const isConfirmActionPending =
		!!confirmAction &&
		pendingAction?.id === confirmAction.container.id &&
		pendingAction?.type === confirmAction.type;

	return (
		<div className="w-full space-y-8">
			<ContainersSummaryCards
				totalContainers={containers.length}
				hostInfo={hostInfo}
				systemUsage={systemUsage}
			/>

			{hostErrors.length > 0 && (
				<output
					className="rounded-lg border border-yellow-500/50 bg-yellow-50 dark:bg-yellow-900/10 p-3 text-sm"
					aria-live="polite"
					aria-atomic="true"
				>
					<p className="font-medium text-yellow-800 dark:text-yellow-200">
						Some Docker hosts are unavailable
					</p>
					<ul className="mt-1 text-yellow-700 dark:text-yellow-300">
						{hostErrors.map((he) => (
							<li key={he.host}>
								{he.host}: {he.message}
							</li>
						))}
					</ul>
				</output>
			)}

			<section className="space-y-4">
				<ContainersToolbar
					searchTerm={searchTerm}
					onSearchChange={handleSearchChange}
					stateFilter={stateFilter}
					onStateFilterChange={handleStateFilterChange}
					availableStates={availableStates}
					hostFilter={hostFilter}
					onHostFilterChange={handleHostFilterChange}
					availableHosts={hosts}
					sortDirection={sortDirection}
					onSortDirectionChange={handleSortDirectionChange}
					sortBy={sortBy}
					onSortByChange={handleSortByChange}
					groupBy={groupBy}
					onGroupByChange={handleGroupByChange}
					statsInterval={statsInterval}
					onStatsIntervalChange={handleStatsIntervalChange}
					dateRange={dateRange}
					onDateRangeChange={handleDateRangeChange}
					onDateRangeClear={handleDateRangeClear}
					onRefresh={refetch}
					isFetching={isFetching}
				/>

				<ContainersStateSummary stateCounts={stateCounts} />

				{selectedContainerIds.length > 0 && (
					<div className="flex items-center justify-between rounded-lg border bg-muted/40 p-4">
						<p className="text-sm font-medium">
							{selectedContainerIds.length} selected
						</p>
						<div className="flex items-center gap-3 text-sm">
							<button type="button" onClick={() => setSelectedContainerIds([])}>
								Clear
							</button>
							<button
								type="button"
								disabled={isReadOnly}
								onClick={() => handleBatchAction("start")}
							>
								Start
							</button>
							<button
								type="button"
								disabled={isReadOnly}
								onClick={() => handleBatchAction("stop")}
							>
								Stop
							</button>
							<button
								type="button"
								disabled={isReadOnly}
								onClick={() => handleBatchAction("restart")}
							>
								Restart
							</button>
							<button
								type="button"
								className="text-destructive"
								disabled={isReadOnly}
								onClick={() => handleBatchAction("remove")}
							>
								Delete
							</button>
						</div>
					</div>
				)}

				<ContainersTable
					isLoading={isLoading}
					isError={isError}
					error={error}
					groupBy={groupBy}
					filteredContainers={filteredContainers}
					groupedItems={groupedItems}
					pageItems={pageItems}
					pendingAction={pendingAction}
					isReadOnly={isReadOnly}
					expandedGroups={expandedGroups}
					selectedIds={selectedContainerIds}
					statsInterval={statsInterval}
					onToggleSelect={handleToggleSelect}
					onSelectAll={handleSelectAll}
					onToggleGroup={handleToggleGroup}
					onStart={handleStartContainer}
					onStop={handleStopContainer}
					onRestart={handleRestartContainer}
					onDelete={handleDeleteContainer}
					onViewLogs={handleViewLogs}
					onViewStats={handleViewStats}
					onRetry={() => {
						void refetch();
					}}
				/>

				<ContainersPagination
					totalItems={filteredContainers.length}
					startIndex={startIndex}
					endIndex={endIndex}
					page={page}
					totalPages={totalPages}
					pageSize={pageSize}
					onPageChange={handlePageChange}
					onPageSizeChange={handlePageSizeChange}
				/>
			</section>

			<AlertDialog
				open={Boolean(confirmAction)}
				onOpenChange={handleConfirmDialogOpenChange}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>{confirmActionTitle}</AlertDialogTitle>
						<AlertDialogDescription>
							{confirmActionDescription}
						</AlertDialogDescription>
					</AlertDialogHeader>
					{confirmAction && (
						<div className="space-y-2">
							<div className="text-sm font-medium text-muted-foreground">
								Container Details
							</div>
							<div className="rounded-md border bg-muted/30 p-3 space-y-2">
								<div className="flex items-start justify-between gap-4">
									<span className="text-xs text-muted-foreground">Name</span>
									<span className="text-sm font-medium text-right">
										{formatContainerName(confirmAction.container.names)}
									</span>
								</div>
								<div className="flex items-start justify-between gap-4">
									<span className="text-xs text-muted-foreground">Image</span>
									<span className="text-sm font-mono text-right break-all">
										{confirmAction.container.image}
									</span>
								</div>
								<div className="flex items-start justify-between gap-4">
									<span className="text-xs text-muted-foreground">ID</span>
									<span className="text-sm font-mono text-right break-all">
										{confirmAction.container.id.slice(0, 12)}
									</span>
								</div>
							</div>
						</div>
					)}
					<AlertDialogFooter>
						<AlertDialogCancel disabled={isConfirmActionPending}>
							Cancel
						</AlertDialogCancel>
						<AlertDialogAction
							className={`flex items-center gap-2 ${
								confirmAction?.type === "remove"
									? "bg-destructive text-white hover:bg-destructive/90"
									: ""
							}`}
							onClick={() => {
								void handleConfirmAction();
							}}
							disabled={isConfirmActionPending}
						>
							{isConfirmActionPending && <Spinner className="size-4" />}
							{confirmActionButtonLabel}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>

			<ContainersLogsSheet
				container={selectedContainer}
				isOpen={isLogsSheetOpen}
				isReadOnly={isReadOnly}
				onOpenChange={handleLogsSheetOpenChange}
				onContainerRecreated={handleContainerRecreated}
			/>

			<ContainerDetailsSheet
				container={detailsContainer}
				host={detailsContainer?.host ?? ""}
				isOpen={isDetailsSheetOpen}
				onOpenChange={handleDetailsSheetOpenChange}
				isReadOnly={isReadOnly}
			/>
		</div>
	);
}
