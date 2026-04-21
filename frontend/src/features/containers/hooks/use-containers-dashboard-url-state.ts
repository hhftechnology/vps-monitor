import {
	createParser,
	parseAsArrayOf,
	parseAsInteger,
	parseAsIsoDateTime,
	parseAsString,
	useQueryStates,
} from "nuqs";
import { useCallback, useMemo } from "react";

import type { DateRange } from "react-day-picker";
import type {
	GroupByOption,
	SortColumn,
	SortDirection,
	StatsInterval,
} from "../components/container-utils";

// Custom parser for SortDirection
const parseAsSortDirection = createParser({
	parse: (value): SortDirection | null => {
		if (value === "asc" || value === "desc") {
			return value;
		}
		return null;
	},
	serialize: (value: SortDirection) => value,
});

// Custom parser for GroupByOption
const parseAsGroupBy = createParser({
	parse: (value): GroupByOption | null => {
		if (value === "none" || value === "compose") {
			return value;
		}
		return null;
	},
	serialize: (value: GroupByOption) => value,
});

const parseAsStatsInterval = createParser({
	parse: (value): StatsInterval | null => {
		if (value === "1h" || value === "12h") {
			return value;
		}
		return null;
	},
	serialize: (value: StatsInterval) => value,
});

const parseAsSortColumn = createParser({
	parse: (value): SortColumn | null => {
		if (["name", "state", "uptime", "created", "cpu", "ram"].includes(value)) {
			return value as SortColumn;
		}
		return null;
	},
	serialize: (value: SortColumn) => value,
});

// Search params configuration with defaults
const searchParamsConfig = {
	search: parseAsString.withDefault(""),
	state: parseAsString.withDefault("all"),
	host: parseAsString.withDefault("all"),
	sort: parseAsSortDirection.withDefault("desc" as SortDirection),
	sortBy: parseAsSortColumn.withDefault("created" as SortColumn),
	group: parseAsGroupBy.withDefault("none" as GroupByOption),
	interval: parseAsStatsInterval.withDefault("1h" as StatsInterval),
	page: parseAsInteger.withDefault(1),
	pageSize: parseAsInteger.withDefault(10),
	from: parseAsIsoDateTime,
	to: parseAsIsoDateTime,
	expanded: parseAsArrayOf(parseAsString).withDefault([]),
};

export function useContainersDashboardUrlState() {
	const [params, setParams] = useQueryStates(searchParamsConfig, {
		history: "replace",
	});

	const {
		search: searchTerm,
		state: stateFilter,
		host: hostFilter,
		sort: sortDirection,
		sortBy,
		group: groupBy,
		interval: statsInterval,
		page,
		pageSize,
		from,
		to,
		expanded: expandedGroups,
	} = params;

	// Convert from/to into DateRange format
	// Supports open-ended ranges: from without to, to without from, or both
	const dateRange = useMemo((): DateRange | undefined => {
		if (!from && !to) {
			return undefined;
		}

		return { from: from ?? undefined, to: to ?? undefined };
	}, [from, to]);

	const setSearchTerm = useCallback(
		(value: string) => {
			setParams({
				search: value,
				page: 1,
			});
		},
		[setParams],
	);

	const setStateFilter = useCallback(
		(value: string) => {
			const normalized = value || "all";
			setParams({
				state: normalized,
				page: 1,
			});
		},
		[setParams],
	);

	const setHostFilter = useCallback(
		(value: string) => {
			const normalized = value || "all";
			setParams({
				host: normalized,
				page: 1,
			});
		},
		[setParams],
	);

	const setSortDirection = useCallback(
		(value: SortDirection) => {
			setParams({
				sort: value,
			});
		},
		[setParams],
	);

	const setSortBy = useCallback(
		(value: SortColumn) => {
			setParams({
				sortBy: value,
			});
		},
		[setParams],
	);

	const setGroupBy = useCallback(
		(value: GroupByOption) => {
			setParams({
				group: value,
				page: 1,
			});
		},
		[setParams],
	);

	const setStatsInterval = useCallback(
		(value: StatsInterval) => {
			setParams({
				interval: value,
			});
		},
		[setParams],
	);

	const setDateRange = useCallback(
		(range: DateRange | undefined) => {
			setParams({
				from: range?.from ?? null,
				to: range?.to ?? null,
				page: 1,
			});
		},
		[setParams],
	);

	const clearDateRange = useCallback(() => {
		setParams({
			from: null,
			to: null,
			page: 1,
		});
	}, [setParams]);

	const setPage = useCallback(
		(value: number) => {
			setParams({
				page: Math.max(1, Math.floor(value)),
			});
		},
		[setParams],
	);

	const setPageSize = useCallback(
		(value: number) => {
			setParams({
				pageSize: Math.max(1, Math.floor(value)),
				page: 1,
			});
		},
		[setParams],
	);

	const setExpandedGroups = useCallback(
		(value: string[]) => {
			setParams({
				expanded: value,
			});
		},
		[setParams],
	);

	return {
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
		page,
		setPage,
		pageSize,
		setPageSize,
		expandedGroups,
		setExpandedGroups,
	};
}
