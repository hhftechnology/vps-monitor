import { beforeEach, describe, expect, it, vi } from "vitest";

import { restartContainer } from "./container-actions";

const storage = new Map<string, string>();

describe("container-actions", () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		storage.clear();
		vi.stubGlobal("localStorage", {
			getItem: (key: string) => storage.get(key) ?? null,
			setItem: (key: string, value: string) => {
				storage.set(key, value);
			},
			removeItem: (key: string) => {
				storage.delete(key);
			},
		});
	});

	it("marks 202 responses as pending", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn().mockResolvedValue(
				new Response(
					JSON.stringify({
						message: "Container restart initiated",
						status: "pending",
					}),
					{
						status: 202,
						headers: { "Content-Type": "application/json" },
					},
				),
			),
		);

		const result = await restartContainer("abc123", "host-a");
		expect(result).toEqual({
			message: "Container restart initiated",
			isPending: true,
		});
	});
});
