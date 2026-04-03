import { afterEach, describe, expect, it, vi } from "vitest";

import { startScan } from "./start-scan";

vi.mock("@/lib/api-client", () => ({
  authenticatedFetch: vi.fn(),
}));

import { authenticatedFetch } from "@/lib/api-client";

const mockFetch = authenticatedFetch as ReturnType<typeof vi.fn>;

const sampleJob = {
  id: "scan-job-1",
  image_ref: "nginx:latest",
  host: "local",
  scanner: "grype",
  status: "pending",
  created_at: 1700000000,
};

describe("startScan", () => {
  afterEach(() => vi.clearAllMocks());

  it("posts to the scan endpoint and returns the job", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 202,
      json: () => Promise.resolve({ job: sampleJob }),
    } as unknown as Response);

    const result = await startScan({
      imageRef: "nginx:latest",
      host: "local",
      scanner: "grype",
    });

    expect(result).toEqual(sampleJob);
    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain("/api/v1/scan");
    expect(opts?.method).toBe("POST");
  });

  it("serializes params correctly in the request body", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 202,
      json: () => Promise.resolve({ job: sampleJob }),
    } as unknown as Response);

    await startScan({ imageRef: "redis:7", host: "remote-host", scanner: "trivy" });

    const [, opts] = mockFetch.mock.calls[0];
    const body = JSON.parse(opts?.body as string);
    expect(body.imageRef).toBe("redis:7");
    expect(body.host).toBe("remote-host");
    expect(body.scanner).toBe("trivy");
  });

  it("throws on server error", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      text: () => Promise.resolve("scan failed"),
    } as unknown as Response);

    await expect(
      startScan({ imageRef: "nginx:latest", host: "local" })
    ).rejects.toThrow("scan failed");
  });

  it("throws with status fallback when error body is empty", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 503,
      text: () => Promise.resolve(""),
    } as unknown as Response);

    await expect(
      startScan({ imageRef: "nginx:latest", host: "local" })
    ).rejects.toThrow("503");
  });
});