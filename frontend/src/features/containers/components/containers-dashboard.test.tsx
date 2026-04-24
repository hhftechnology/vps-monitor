import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ContainersDashboard } from "./containers-dashboard";

const mockUseContainersQuery = vi.fn();
const mockUseSystemStats = vi.fn();
const mockUseContainersDashboardUrlState = vi.fn();
const mockStartContainer = vi.fn();
const mockStopContainer = vi.fn();
const mockRestartContainer = vi.fn();
const mockRemoveContainer = vi.fn();
const mockContainersLogsSheet = vi.fn((_props: unknown) => (
	<div data-testid="logs-sheet" />
));
const mockContainerDetailsSheet = vi.fn((_props: unknown) => (
	<div data-testid="details-sheet" />
));

vi.mock("../hooks/use-containers-query", () => ({
	useContainersQuery: (...args: unknown[]) => mockUseContainersQuery(...args),
}));

vi.mock("../api/container-actions", () => ({
	startContainer: (...args: unknown[]) => mockStartContainer(...args),
	stopContainer: (...args: unknown[]) => mockStopContainer(...args),
	restartContainer: (...args: unknown[]) => mockRestartContainer(...args),
	removeContainer: (...args: unknown[]) => mockRemoveContainer(...args),
}));

vi.mock("../hooks/use-system-stats", () => ({
	useSystemStats: (...args: unknown[]) => mockUseSystemStats(...args),
}));

vi.mock("../hooks/use-containers-dashboard-url-state", () => ({
	useContainersDashboardUrlState: (...args: unknown[]) =>
		mockUseContainersDashboardUrlState(...args),
}));

vi.mock("./containers-summary-cards", () => ({
	ContainersSummaryCards: () => <div>summary cards</div>,
}));

vi.mock("./containers-toolbar", () => ({
	ContainersToolbar: () => <div>toolbar</div>,
}));

vi.mock("./containers-state-summary", () => ({
	ContainersStateSummary: () => <div>state summary</div>,
}));

vi.mock("./containers-pagination", () => ({
	ContainersPagination: () => <div>pagination</div>,
}));

vi.mock("./containers-table", () => ({
	ContainersTable: ({
		pageItems,
		onToggleSelect,
		onViewLogs,
		onViewStats,
	}: {
		pageItems: Array<{ id: string }>;
		onToggleSelect: (id: string) => void;
		onViewLogs: (container: { id: string }) => void;
		onViewStats: (container: { id: string }) => void;
	}) => (
		<div>
			{pageItems.map((item) => (
				<button
					key={item.id}
					type="button"
					onClick={() => onToggleSelect(item.id)}
				>
					Select {item.id}
				</button>
			))}
			<button type="button" onClick={() => onViewLogs(pageItems[0])}>
				Open logs
			</button>
			<button type="button" onClick={() => onViewStats(pageItems[0])}>
				Open stats
			</button>
		</div>
	),
}));

vi.mock("./containers-logs-sheet", () => ({
	ContainersLogsSheet: (props: unknown) => mockContainersLogsSheet(props),
}));

vi.mock("./container-details-sheet", () => ({
	ContainerDetailsSheet: (props: unknown) => mockContainerDetailsSheet(props),
}));

const container = {
	id: "container-1",
	names: ["/api"],
	image: "ghcr.io/example/api:latest",
	image_id: "sha256:123",
	command: "node server.js",
	created: 1_700_000_000,
	state: "running",
	status: "Up 2 hours",
	host: "local",
	historical_stats: {
		cpu_1h: 12,
		memory_1h: 34,
		cpu_12h: 20,
		memory_12h: 40,
	},
};

const secondContainer = {
	...container,
	id: "container-2",
	names: ["/worker"],
	image: "ghcr.io/example/worker:latest",
};

function renderDashboard() {
	const queryClient = new QueryClient();
	return render(
		<QueryClientProvider client={queryClient}>
			<ContainersDashboard />
		</QueryClientProvider>,
	);
}

describe("ContainersDashboard", () => {
	beforeEach(() => {
		vi.clearAllMocks();
		mockStartContainer.mockResolvedValue({ message: "started" });
		mockStopContainer.mockResolvedValue({ message: "stopped" });
		mockRestartContainer.mockResolvedValue({ message: "restarted" });
		mockRemoveContainer.mockResolvedValue({ message: "removed" });

		mockUseContainersQuery.mockReturnValue({
			data: {
				containers: [container],
				readOnly: false,
				hosts: [{ Name: "local", Host: "unix:///var/run/docker.sock" }],
				hostErrors: [],
			},
			error: null,
			isError: false,
			isFetching: false,
			isLoading: false,
			refetch: vi.fn(),
		});

		mockUseSystemStats.mockReturnValue({
			data: {
				hostInfo: {
					hostname: "vps-1",
					platform: "linux",
					kernelVersion: "6.8.0",
				},
				usage: {
					cpuPercent: 12,
					memoryPercent: 40,
					diskPercent: 55,
				},
			},
		});

		mockUseContainersDashboardUrlState.mockReturnValue({
			searchTerm: "",
			setSearchTerm: vi.fn(),
			stateFilter: "all",
			setStateFilter: vi.fn(),
			hostFilter: "all",
			setHostFilter: vi.fn(),
			sortDirection: "desc",
			setSortDirection: vi.fn(),
			sortBy: "created",
			setSortBy: vi.fn(),
			groupBy: "none",
			setGroupBy: vi.fn(),
			statsInterval: "1h",
			setStatsInterval: vi.fn(),
			dateRange: undefined,
			setDateRange: vi.fn(),
			clearDateRange: vi.fn(),
			pageSize: 10,
			setPageSize: vi.fn(),
			page: 1,
			setPage: vi.fn(),
			expandedGroups: [],
			setExpandedGroups: vi.fn(),
		});
	});

	it("renders the home dashboard without mounting closed overlay sheets", () => {
		renderDashboard();

		expect(screen.getByText("summary cards")).toBeInTheDocument();
		expect(mockContainersLogsSheet).not.toHaveBeenCalled();
		expect(mockContainerDetailsSheet).not.toHaveBeenCalled();
	});

	it("mounts the requested sheet only after the matching action is triggered", () => {
		renderDashboard();

		fireEvent.click(screen.getByRole("button", { name: "Open logs" }));
		expect(mockContainersLogsSheet).toHaveBeenCalledTimes(1);
		expect(screen.getByTestId("logs-sheet")).toBeInTheDocument();
		expect(mockContainerDetailsSheet).not.toHaveBeenCalled();

		fireEvent.click(screen.getByRole("button", { name: "Open stats" }));
		expect(mockContainerDetailsSheet).toHaveBeenCalledTimes(1);
		expect(screen.getByTestId("details-sheet")).toBeInTheDocument();
	});

	it("confirms multi-select stop before executing actions", async () => {
		mockUseContainersQuery.mockReturnValue({
			data: {
				containers: [container, secondContainer],
				readOnly: false,
				hosts: [{ Name: "local", Host: "unix:///var/run/docker.sock" }],
				hostErrors: [],
			},
			error: null,
			isError: false,
			isFetching: false,
			isLoading: false,
			refetch: vi.fn(),
		});

		renderDashboard();

		fireEvent.click(screen.getByRole("button", { name: "Select container-1" }));
		fireEvent.click(screen.getByRole("button", { name: "Select container-2" }));
		fireEvent.click(screen.getByRole("button", { name: "Stop" }));

		expect(mockStopContainer).not.toHaveBeenCalled();

		fireEvent.click(screen.getByRole("button", { name: "Stop Containers" }));

		await waitFor(() => expect(mockStopContainer).toHaveBeenCalledTimes(2));
	});
});
