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

type DashboardUrlParams = {
	search: string;
	state: string;
	host: string;
	sort: SortDirection;
	sortBy: SortColumn;
	group: GroupByOption;
	interval: StatsInterval;
	page: number;
	pageSize: number;
	from: Date | null;
	to: Date | null;
	expanded: string[];
};

function areStringArraysEqual(left: string[], right: string[]) {
	if (left.length !== right.length) {
		return false;
	}

	return left.every((value, index) => value === right[index]);
}

function areDatesEqual(left: Date | null, right: Date | null) {
	if (left === right) {
		return true;
	}

	if (!left || !right) {
		return false;
	}

	return left.getTime() === right.getTime();
}

function isDashboardParamEqual(
	currentValue: DashboardUrlParams[keyof DashboardUrlParams],
	nextValue: DashboardUrlParams[keyof DashboardUrlParams],
) {
	if (Array.isArray(currentValue) && Array.isArray(nextValue)) {
		return areStringArraysEqual(currentValue, nextValue);
	}

	if (
		(currentValue instanceof Date || currentValue === null) &&
		(nextValue instanceof Date || nextValue === null)
	) {
		return areDatesEqual(currentValue, nextValue);
	}

	return currentValue === nextValue;
}

export function hasDashboardParamChanges(
	current: DashboardUrlParams,
	updates: Partial<DashboardUrlParams>,
) {
	return Object.entries(updates).some(([key, value]) => {
		const typedKey = key as keyof DashboardUrlParams;
		return !isDashboardParamEqual(
			current[typedKey],
			value as DashboardUrlParams[keyof DashboardUrlParams],
		);
	});
}

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

	const updateParams = useCallback(
		(updates: Partial<DashboardUrlParams>) => {
			if (hasDashboardParamChanges(params as DashboardUrlParams, updates)) {
				setParams(updates);
			}
		},
		[params, setParams],
	);

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
			updateParams({
				search: value,
				page: 1,
			});
		},
		[updateParams],
	);

	const setStateFilter = useCallback(
		(value: string) => {
			const normalized = value || "all";
			updateParams({
				state: normalized,
				page: 1,
			});
		},
		[updateParams],
	);

	const setHostFilter = useCallback(
		(value: string) => {
			const normalized = value || "all";
			updateParams({
				host: normalized,
				page: 1,
			});
		},
		[updateParams],
	);

	const setSortDirection = useCallback(
		(value: SortDirection) => {
			updateParams({
				sort: value,
			});
		},
		[updateParams],
	);

	const setSortBy = useCallback(
		(value: SortColumn) => {
			updateParams({
				sortBy: value,
			});
		},
		[updateParams],
	);

	const setGroupBy = useCallback(
		(value: GroupByOption) => {
			updateParams({
				group: value,
				page: 1,
			});
		},
		[updateParams],
	);

	const setStatsInterval = useCallback(
		(value: StatsInterval) => {
			updateParams({
				interval: value,
			});
		},
		[updateParams],
	);

	const setDateRange = useCallback(
		(range: DateRange | undefined) => {
			updateParams({
				from: range?.from ?? null,
				to: range?.to ?? null,
				page: 1,
			});
		},
		[updateParams],
	);

	const clearDateRange = useCallback(() => {
		updateParams({
			from: null,
			to: null,
			page: 1,
		});
	}, [updateParams]);

	const setPage = useCallback(
		(value: number) => {
			updateParams({
				page: Math.max(1, Math.floor(value)),
			});
		},
		[updateParams],
	);

	const setPageSize = useCallback(
		(value: number) => {
			updateParams({
				pageSize: Math.max(1, Math.floor(value)),
				page: 1,
			});
		},
		[updateParams],
	);

	const setExpandedGroups = useCallback(
		(value: string[]) => {
			updateParams({
				expanded: value,
			});
		},
		[updateParams],
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
