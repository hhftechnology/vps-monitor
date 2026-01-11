import {
  createParser,
  parseAsInteger,
  parseAsIsoDateTime,
  parseAsString,
  useQueryStates
} from "nuqs";
import { useCallback, useMemo } from "react";

import type { DateRange } from "react-day-picker";
import type {
  GroupByOption,
  SortDirection,
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

// Search params configuration with defaults
const searchParamsConfig = {
  search: parseAsString.withDefault(""),
  state: parseAsString.withDefault("all"),
  host: parseAsString.withDefault("all"),
  sort: parseAsSortDirection.withDefault("desc" as SortDirection),
  group: parseAsGroupBy.withDefault("none" as GroupByOption),
  page: parseAsInteger.withDefault(1),
  pageSize: parseAsInteger.withDefault(10),
  from: parseAsIsoDateTime,
  to: parseAsIsoDateTime,
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
    group: groupBy,
    page,
    pageSize,
    from,
    to,
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
    [setParams]
  );

  const setStateFilter = useCallback(
    (value: string) => {
      const normalized = value || "all";
      setParams({
        state: normalized,
        page: 1,
      });
    },
    [setParams]
  );

  const setHostFilter = useCallback(
    (value: string) => {
      const normalized = value || "all";
      setParams({
        host: normalized,
        page: 1,
      });
    },
    [setParams]
  );

  const setSortDirection = useCallback(
    (value: SortDirection) => {
      setParams({
        sort: value,
      });
    },
    [setParams]
  );

  const setGroupBy = useCallback(
    (value: GroupByOption) => {
      setParams({
        group: value,
        page: 1,
      });
    },
    [setParams]
  );

  const setDateRange = useCallback(
    (range: DateRange | undefined) => {
      setParams({
        from: range?.from ?? null,
        to: range?.to ?? null,
        page: 1,
      });
    },
    [setParams]
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
    [setParams]
  );

  const setPageSize = useCallback(
    (value: number) => {
      setParams({
        pageSize: Math.max(1, Math.floor(value)),
        page: 1,
      });
    },
    [setParams]
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
    groupBy,
    setGroupBy,
    dateRange,
    setDateRange,
    clearDateRange,
    page,
    setPage,
    pageSize,
    setPageSize,
  };
}
