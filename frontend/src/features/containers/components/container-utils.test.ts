import { describe, expect, it } from "vitest";

import { groupByCompose } from "./container-utils";

describe("groupByCompose", () => {
	it("sorts compose groups by newest container when descending", () => {
		const groups = groupByCompose(
			[
				{
					id: "1",
					names: ["/web-1"],
					image: "img",
					image_id: "sha",
					command: "run",
					created: 100,
					state: "running",
					status: "up",
					host: "host-a",
					labels: { "com.docker.compose.project": "project-old" },
				},
				{
					id: "2",
					names: ["/api-1"],
					image: "img",
					image_id: "sha",
					command: "run",
					created: 200,
					state: "running",
					status: "up",
					host: "host-a",
					labels: { "com.docker.compose.project": "project-new" },
				},
			],
			"desc",
		);

		expect(groups.map((group) => group.project)).toEqual([
			"project-new",
			"project-old",
		]);
	});
});
