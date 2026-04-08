import { afterEach, describe, expect, it, vi } from "vitest";

import { downloadSBOMHistoryFile } from "./get-sbom-history";

vi.mock("@/lib/api-client", () => ({
  authenticatedFetch: vi.fn(),
}));

import { authenticatedFetch } from "@/lib/api-client";

const mockFetch = authenticatedFetch as ReturnType<typeof vi.fn>;

describe("downloadSBOMHistoryFile", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("uses the server-provided filename from Content-Disposition", async () => {
    const blob = new Blob(["{}"], { type: "application/json" });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      headers: new Headers({
        "Content-Disposition": 'attachment; filename="scan-result.json"',
      }),
      blob: () => Promise.resolve(blob),
    } as unknown as Response);

    const result = await downloadSBOMHistoryFile("sbom-123");

    expect(result.blob).toBe(blob);
    expect(result.filename).toBe("scan-result.json");
  });

  it("falls back to the default filename when the header is missing", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      headers: new Headers(),
      blob: () => Promise.resolve(new Blob(["{}"], { type: "application/json" })),
    } as unknown as Response);

    const result = await downloadSBOMHistoryFile("sbom-123");

    expect(result.filename).toBe("sbom-sbom-123.json");
  });
});
