import { describe, expect, it } from "vitest";

import { hasDashboardParamChanges } from "./use-containers-dashboard-url-state";

const baseParams = {
	search: "",
	state: "all",
	host: "all",
	sort: "desc" as const,
	sortBy: "created" as const,
	group: "none" as const,
	interval: "1h" as const,
	page: 1,
	pageSize: 10,
	from: null,
	to: null,
	expanded: ["project-a"],
};

describe("hasDashboardParamChanges", () => {
	it("returns false when updates do not change the current params", () => {
		expect(
			hasDashboardParamChanges(baseParams, {
				search: "",
				page: 1,
				expanded: ["project-a"],
			}),
		).toBe(false);
	});

	it("compares date values by timestamp rather than object identity", () => {
		const current = {
			...baseParams,
			from: new Date("2026-04-22T00:00:00.000Z"),
			to: new Date("2026-04-23T00:00:00.000Z"),
		};

		expect(
			hasDashboardParamChanges(current, {
				from: new Date("2026-04-22T00:00:00.000Z"),
				to: new Date("2026-04-23T00:00:00.000Z"),
			}),
		).toBe(false);
	});

	it("returns true when any param value actually changes", () => {
		expect(
			hasDashboardParamChanges(baseParams, {
				host: "prod-1",
			}),
		).toBe(true);
	});
});
