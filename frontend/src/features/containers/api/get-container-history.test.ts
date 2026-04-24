import { authenticatedFetch } from "@/lib/api-client";
import { afterEach, describe, expect, it, vi } from "vitest";

import { getContainerHistory } from "./get-container-history";

vi.mock("@/lib/api-client", () => ({
	authenticatedFetch: vi.fn(),
}));

const mockFetch = authenticatedFetch as ReturnType<typeof vi.fn>;

describe("getContainerHistory", () => {
	afterEach(() => vi.clearAllMocks());

	it("normalizes missing samples to an empty array", async () => {
		mockFetch.mockResolvedValueOnce({
			ok: true,
			json: () =>
				Promise.resolve({
					cpu_1h: 1,
					memory_1h: 2,
					cpu_12h: 3,
					memory_12h: 4,
					has_data: true,
				}),
		} as unknown as Response);

		const history = await getContainerHistory("container-1", "local");

		expect(history.samples).toEqual([]);
	});
});
