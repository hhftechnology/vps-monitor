import { afterEach, describe, expect, it, vi } from "vitest";

import { startBulkScan } from "./start-bulk-scan";

vi.mock("@/lib/api-client", () => ({
  authenticatedFetch: vi.fn(),
}));

import { authenticatedFetch } from "@/lib/api-client";

const mockFetch = authenticatedFetch as ReturnType<typeof vi.fn>;

const sampleBulkJob = {
  id: "bulk-job-1",
  jobs: [],
  total_images: 0,
  completed: 0,
  failed: 0,
  status: "pending",
  created_at: 1700000000,
};

describe("startBulkScan", () => {
  afterEach(() => vi.clearAllMocks());

  it("posts to the bulk scan endpoint and returns the bulk job", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 202,
      json: () => Promise.resolve({ job: sampleBulkJob }),
    } as unknown as Response);

    const result = await startBulkScan({ scanner: "grype" });

    expect(result).toEqual(sampleBulkJob);
    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain("/api/v1/scan/bulk");
    expect(opts?.method).toBe("POST");
  });

  it("serializes hosts filter in the request body", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 202,
      json: () => Promise.resolve({ job: sampleBulkJob }),
    } as unknown as Response);

    await startBulkScan({ scanner: "trivy", hosts: ["host-a", "host-b"] });

    const [, opts] = mockFetch.mock.calls[0];
    const body = JSON.parse(opts?.body as string);
    expect(body.hosts).toEqual(["host-a", "host-b"]);
    expect(body.scanner).toBe("trivy");
  });

  it("throws on server error", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      text: () => Promise.resolve("bulk scan failed"),
    } as unknown as Response);

    await expect(startBulkScan({})).rejects.toThrow("bulk scan failed");
  });

  it("throws with status fallback when error body is empty", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      text: () => Promise.resolve(""),
    } as unknown as Response);

    await expect(startBulkScan({ scanner: "grype" })).rejects.toThrow("500");
  });
});