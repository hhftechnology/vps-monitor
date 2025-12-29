import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";
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
import { useContainersDashboardUrlState } from "../hooks/use-containers-dashboard-url-state";
import { useContainersQuery } from "../hooks/use-containers-query";
import { useSystemStats } from "../hooks/use-system-stats";

import {
  formatContainerName,
  getInitialStateCounts,
  groupByCompose,
} from "./container-utils";
import { ContainersLogsSheet } from "./containers-logs-sheet";
import { ContainersPagination } from "./containers-pagination";
import { ContainersStateSummary } from "./containers-state-summary";
import { ContainersSummaryCards } from "./containers-summary-cards";
import { ContainersTable } from "./containers-table";
import { ContainersToolbar } from "./containers-toolbar";

import type { DateRange } from "react-day-picker";
import type { GetContainersResponse } from "../api/get-containers";
import type { ContainerInfo } from "../types";
import type {
  ContainerActionType,
  GroupByOption,
  SortDirection,
} from "./container-utils";

export function ContainersDashboard() {
  const queryClient = useQueryClient();
  const { data, error, isError, isFetching, isLoading, refetch } =
    useContainersQuery();
  const { data: systemStats } = useSystemStats();

  const containers = data?.containers ?? [];
  const isReadOnly = data?.readOnly ?? false;
  const hosts = data?.hosts ?? [];

  const hostInfo = useMemo(
    () => ({
      hostname: systemStats?.hostInfo.hostname ?? "Loading...",
      os: systemStats?.hostInfo.platform ?? "Unknown",
      kernel: systemStats?.hostInfo.kernelVersion ?? "Unknown",
    }),
    [systemStats]
  );

  const systemUsage = useMemo(
    () => ({
      cpu: Math.round(systemStats?.usage.cpuPercent ?? 0),
      memory: Math.round(systemStats?.usage.memoryPercent ?? 0),
    }),
    [systemStats]
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
    groupBy,
    setGroupBy,
    dateRange,
    setDateRange,
    clearDateRange,
    pageSize,
    setPageSize,
    page,
    setPage,
  } = useContainersDashboardUrlState();
  const [selectedContainer, setSelectedContainer] =
    useState<ContainerInfo | null>(null);
  const [isLogsSheetOpen, setIsLogsSheetOpen] = useState(false);
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
      options: { includeStateFilter?: boolean } = {}
    ) => {
      const matchesSearch =
        !normalizedSearch ||
        container.id.toLowerCase().startsWith(normalizedSearch) ||
        container.image.toLowerCase().includes(normalizedSearch) ||
        container.names.some((name) =>
          name.toLowerCase().includes(normalizedSearch)
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

  const filteredContainers = useMemo(() => {
    const filtered = containers.filter((container) =>
      matchesFilters(container, { includeStateFilter: true })
    );

    return filtered.sort((a, b) =>
      sortDirection === "desc" ? b.created - a.created : a.created - b.created
    );
  }, [containers, matchesFilters, sortDirection]);

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
    return groupByCompose(pageItems);
  }, [pageItems, groupBy]);

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
    container: ContainerInfo
  ) => {
    setPendingAction({ id: container.id, type: actionType });
    try {
      let message = "";
      switch (actionType) {
        case "start":
          message = await startContainer(container.id, container.host);
          break;
        case "stop":
          message = await stopContainer(container.id, container.host);
          break;
        case "restart":
          message = await restartContainer(container.id, container.host);
          break;
        case "remove":
          message = await removeContainer(container.id, container.host);
          break;
        default:
          return;
      }
      if (message) {
        toast.success(message);
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

  const handleGroupByChange = (value: GroupByOption) => {
    setGroupBy(value);
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

  const handleContainerRecreated = async (newContainerId: string) => {
    await queryClient.refetchQueries({
      queryKey: ["containers"],
      exact: false,
    });

    const updatedData = queryClient.getQueryData<GetContainersResponse>([
      "containers",
    ]);
    const newContainer = updatedData?.containers?.find(
      (c) => c.id === newContainerId
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
          groupBy={groupBy}
          onGroupByChange={handleGroupByChange}
          dateRange={dateRange}
          onDateRangeChange={handleDateRangeChange}
          onDateRangeClear={handleDateRangeClear}
          onRefresh={refetch}
          isFetching={isFetching}
        />

        <ContainersStateSummary stateCounts={stateCounts} />

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
          onStart={handleStartContainer}
          onStop={handleStopContainer}
          onRestart={handleRestartContainer}
          onDelete={handleDeleteContainer}
          onViewLogs={handleViewLogs}
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
    </div>
  );
}
