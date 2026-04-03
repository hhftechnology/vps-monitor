import { afterEach, describe, expect, it, vi } from "vitest";

import { cancelScanJob, getScanJob, getScanJobs } from "./get-scan-jobs";

vi.mock("@/lib/api-client", () => ({
  authenticatedFetch: vi.fn(),
}));

import { authenticatedFetch } from "@/lib/api-client";

const mockFetch = authenticatedFetch as ReturnType<typeof vi.fn>;

function okResponse(body: unknown): Response {
  return {
    ok: true,
    status: 200,
    json: () => Promise.resolve(body),
    text: () => Promise.resolve(JSON.stringify(body)),
  } as unknown as Response;
}

function errResponse(status: number, msg: string): Response {
  return {
    ok: false,
    status,
    text: () => Promise.resolve(msg),
  } as unknown as Response;
}

const sampleJob = {
  id: "job-1",
  image_ref: "nginx:latest",
  host: "local",
  scanner: "grype",
  status: "complete",
  created_at: 1700000000,
};

const sampleBulkJob = {
  id: "bulk-1",
  jobs: [sampleJob],
  total_images: 1,
  completed: 1,
  failed: 0,
  status: "complete",
  created_at: 1700000000,
};

describe("getScanJobs", () => {
  afterEach(() => vi.clearAllMocks());

  it("returns jobs and bulkJobs arrays", async () => {
    mockFetch.mockResolvedValueOnce(
      okResponse({ jobs: [sampleJob], bulkJobs: [sampleBulkJob] })
    );

    const result = await getScanJobs();

    expect(result.jobs).toHaveLength(1);
    expect(result.bulkJobs).toHaveLength(1);
  });

  it("throws on server error", async () => {
    mockFetch.mockResolvedValueOnce(errResponse(500, "server error"));

    await expect(getScanJobs()).rejects.toThrow("server error");
  });

  it("throws with status when error body is empty", async () => {
    mockFetch.mockResolvedValueOnce(errResponse(503, ""));

    await expect(getScanJobs()).rejects.toThrow("503");
  });
});

describe("getScanJob", () => {
  afterEach(() => vi.clearAllMocks());

  it("returns a regular job by ID", async () => {
    mockFetch.mockResolvedValueOnce(okResponse({ job: sampleJob }));

    const result = await getScanJob("job-1");

    expect(result.job).toEqual(sampleJob);
    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain("job-1");
  });

  it("returns a bulkJob when found", async () => {
    mockFetch.mockResolvedValueOnce(okResponse({ bulkJob: sampleBulkJob }));

    const result = await getScanJob("bulk-1");

    expect(result.bulkJob).toEqual(sampleBulkJob);
  });

  it("throws on 404", async () => {
    mockFetch.mockResolvedValueOnce(errResponse(404, "not found"));

    await expect(getScanJob("ghost")).rejects.toThrow("not found");
  });
});

describe("cancelScanJob", () => {
  afterEach(() => vi.clearAllMocks());

  it("sends DELETE request for the given job ID", async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, status: 200 } as Response);

    await cancelScanJob("job-1");

    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain("job-1");
    expect(opts?.method).toBe("DELETE");
  });

  it("throws on non-ok response", async () => {
    mockFetch.mockResolvedValueOnce(errResponse(404, "job not found"));

    await expect(cancelScanJob("ghost")).rejects.toThrow("job not found");
  });
});