import { Link, useNavigate } from "@tanstack/react-router";
import {
	CalendarIcon,
	ChevronDownIcon,
	LogOutIcon,
	RefreshCcwIcon,
	SettingsIcon,
	XIcon,
} from "lucide-react";
import type { DateRange } from "react-day-picker";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuRadioGroup,
	DropdownMenuRadioItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover";
import {
	Tooltip,
	TooltipContent,
	TooltipTrigger,
} from "@/components/ui/tooltip";
import { useAuth } from "@/contexts/auth-context";
import type { DockerHost } from "../types";
import type {
	GroupByOption,
	SortColumn,
	SortDirection,
	StatsInterval,
} from "./container-utils";
import { toTitleCase } from "./container-utils";

interface ContainersToolbarProps {
	searchTerm: string;
	onSearchChange: (value: string) => void;
	stateFilter: string;
	onStateFilterChange: (value: string) => void;
	availableStates: string[];
	hostFilter: string;
	onHostFilterChange: (value: string) => void;
	availableHosts: DockerHost[];
	sortDirection: SortDirection;
	onSortDirectionChange: (direction: SortDirection) => void;
	sortBy: SortColumn;
	onSortByChange: (value: SortColumn) => void;
	groupBy: GroupByOption;
	onGroupByChange: (value: GroupByOption) => void;
	statsInterval: StatsInterval;
	onStatsIntervalChange: (value: StatsInterval) => void;
	dateRange: DateRange | undefined;
	onDateRangeChange: (range: DateRange | undefined) => void;
	onDateRangeClear: () => void;
	onRefresh: () => void;
	isFetching: boolean;
}

export function ContainersToolbar({
	searchTerm,
	onSearchChange,
	stateFilter,
	onStateFilterChange,
	availableStates,
	hostFilter,
	onHostFilterChange,
	availableHosts,
	sortDirection,
	onSortDirectionChange,
	sortBy,
	onSortByChange,
	groupBy,
	onGroupByChange,
	statsInterval,
	onStatsIntervalChange,
	dateRange,
	onDateRangeChange,
	onDateRangeClear,
	onRefresh,
	isFetching,
}: ContainersToolbarProps) {
	const { logout, user, isAuthEnabled } = useAuth();
	const navigate = useNavigate();

	const handleLogout = () => {
		logout();
		navigate({ to: "/login" });
	};

	const sortLabels: Record<SortColumn, string> = {
		name: "Name",
		state: "State",
		uptime: "Uptime",
		created: "Created",
		cpu: "CPU",
		ram: "RAM",
	};

	const renderDateRange = () => {
		if (!dateRange?.from) {
			return <span>Date range</span>;
		}

		if (dateRange.to) {
			const from = dateRange.from.toLocaleDateString("en-US", {
				month: "short",
				day: "numeric",
			});
			const to = dateRange.to.toLocaleDateString("en-US", {
				month: "short",
				day: "numeric",
			});
			return (
				<>
					{from} - {to}
				</>
			);
		}

		return dateRange.from.toLocaleDateString("en-US", {
			month: "short",
			day: "numeric",
		});
	};

	return (
		<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
			<Input
				type="search"
				value={searchTerm}
				onChange={(event) => onSearchChange(event.target.value)}
				placeholder="Search containers..."
				className="sm:max-w-sm"
			/>
			<div className="flex items-center gap-2">
				<DropdownMenu>
					<DropdownMenuTrigger asChild>
						<Button variant="outline" size="sm" className="h-9">
							{hostFilter === "all" ? "All hosts" : hostFilter}
							<ChevronDownIcon className="ml-2 size-4" />
						</Button>
					</DropdownMenuTrigger>
					<DropdownMenuContent align="end">
						<DropdownMenuRadioGroup
							value={hostFilter}
							onValueChange={onHostFilterChange}
						>
							<DropdownMenuRadioItem value="all">
								All hosts
							</DropdownMenuRadioItem>
							{availableHosts.map((host) => (
								<DropdownMenuRadioItem key={host.Name} value={host.Name}>
									{host.Name}
								</DropdownMenuRadioItem>
							))}
						</DropdownMenuRadioGroup>
					</DropdownMenuContent>
				</DropdownMenu>

				<DropdownMenu>
					<DropdownMenuTrigger asChild>
						<Button variant="outline" size="sm" className="h-9">
							{stateFilter === "all" ? "All states" : toTitleCase(stateFilter)}
							<ChevronDownIcon className="ml-2 size-4" />
						</Button>
					</DropdownMenuTrigger>
					<DropdownMenuContent align="end">
						<DropdownMenuRadioGroup
							value={stateFilter}
							onValueChange={onStateFilterChange}
						>
							<DropdownMenuRadioItem value="all">
								All states
							</DropdownMenuRadioItem>
							{availableStates.map((state) => (
								<DropdownMenuRadioItem key={state} value={state}>
									{toTitleCase(state)}
								</DropdownMenuRadioItem>
							))}
						</DropdownMenuRadioGroup>
					</DropdownMenuContent>
				</DropdownMenu>

				<DropdownMenu>
					<DropdownMenuTrigger asChild>
						<Button variant="outline" size="sm" className="h-9">
							Sort: {sortLabels[sortBy]}
							<ChevronDownIcon className="ml-2 size-4" />
						</Button>
					</DropdownMenuTrigger>
					<DropdownMenuContent align="end">
						<DropdownMenuRadioGroup
							value={sortBy}
							onValueChange={(value) => onSortByChange(value as SortColumn)}
						>
							<DropdownMenuRadioItem value="name">Name</DropdownMenuRadioItem>
							<DropdownMenuRadioItem value="state">State</DropdownMenuRadioItem>
							<DropdownMenuRadioItem value="uptime">
								Uptime
							</DropdownMenuRadioItem>
							<DropdownMenuRadioItem value="created">
								Created
							</DropdownMenuRadioItem>
							<DropdownMenuRadioItem value="cpu">CPU</DropdownMenuRadioItem>
							<DropdownMenuRadioItem value="ram">RAM</DropdownMenuRadioItem>
						</DropdownMenuRadioGroup>
					</DropdownMenuContent>
				</DropdownMenu>

				<DropdownMenu>
					<DropdownMenuTrigger asChild>
						<Button variant="outline" size="sm" className="h-9">
							{sortDirection === "desc" ? "Desc" : "Asc"}
							<ChevronDownIcon className="ml-2 size-4" />
						</Button>
					</DropdownMenuTrigger>
					<DropdownMenuContent align="end">
						<DropdownMenuRadioGroup
							value={sortDirection}
							onValueChange={(value) =>
								onSortDirectionChange(value as SortDirection)
							}
						>
							<DropdownMenuRadioItem value="desc">
								Descending
							</DropdownMenuRadioItem>
							<DropdownMenuRadioItem value="asc">
								Ascending
							</DropdownMenuRadioItem>
						</DropdownMenuRadioGroup>
					</DropdownMenuContent>
				</DropdownMenu>

				<DropdownMenu>
					<DropdownMenuTrigger asChild>
						<Button variant="outline" size="sm" className="h-9">
							{groupBy === "compose" ? "By project" : "No grouping"}
							<ChevronDownIcon className="ml-2 size-4" />
						</Button>
					</DropdownMenuTrigger>
					<DropdownMenuContent align="end">
						<DropdownMenuRadioGroup
							value={groupBy}
							onValueChange={(value) => onGroupByChange(value as GroupByOption)}
						>
							<DropdownMenuRadioItem value="none">
								No grouping
							</DropdownMenuRadioItem>
							<DropdownMenuRadioItem value="compose">
								By compose project
							</DropdownMenuRadioItem>
						</DropdownMenuRadioGroup>
					</DropdownMenuContent>
				</DropdownMenu>

				<div className="flex items-center rounded-md border bg-background p-1">
					<Button
						variant={statsInterval === "1h" ? "secondary" : "ghost"}
						size="sm"
						className="h-7 px-2 text-xs"
						onClick={() => onStatsIntervalChange("1h")}
					>
						1h
					</Button>
					<Button
						variant={statsInterval === "12h" ? "secondary" : "ghost"}
						size="sm"
						className="h-7 px-2 text-xs"
						onClick={() => onStatsIntervalChange("12h")}
					>
						12h
					</Button>
				</div>

				<Popover>
					<PopoverTrigger asChild>
						<Button
							variant={dateRange?.from ? "default" : "outline"}
							size="sm"
							className="h-9 justify-start text-left font-normal"
						>
							<CalendarIcon className="mr-2 size-4" />
							{renderDateRange()}
							{dateRange?.from && (
								<XIcon
									className="ml-2 size-4 hover:text-destructive"
									onClick={(event) => {
										event.stopPropagation();
										onDateRangeClear();
									}}
								/>
							)}
						</Button>
					</PopoverTrigger>
					<PopoverContent className="w-auto p-0" align="end">
						<Calendar
							mode="range"
							defaultMonth={dateRange?.from}
							selected={dateRange}
							onSelect={onDateRangeChange}
							numberOfMonths={2}
						/>
					</PopoverContent>
				</Popover>

				<Button
					variant="ghost"
					size="sm"
					onClick={onRefresh}
					className="h-9 shrink-0"
				>
					<RefreshCcwIcon
						className={`size-4 ${isFetching ? "animate-spin" : ""}`}
					/>
				</Button>

				<Tooltip>
					<TooltipTrigger asChild>
						<Button variant="ghost" size="sm" className="h-9 shrink-0" asChild>
							<Link to="/settings" aria-label="Settings">
								<SettingsIcon className="size-4" />
							</Link>
						</Button>
					</TooltipTrigger>
					<TooltipContent>Settings</TooltipContent>
				</Tooltip>

				{isAuthEnabled && (
					<Tooltip>
						<TooltipTrigger asChild>
							<Button
								variant="ghost"
								size="sm"
								onClick={handleLogout}
								className="h-9 shrink-0"
							>
								<LogOutIcon className="size-4" />
							</Button>
						</TooltipTrigger>
						<TooltipContent>
							Logout {user?.username ? `(${user.username})` : ""}
						</TooltipContent>
					</Tooltip>
				)}
			</div>
		</div>
	);
}
