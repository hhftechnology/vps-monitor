import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { downloadSBOM, generateSBOM, getSBOMDownloadURL, getSBOMJob } from "./generate-sbom";

// Mock the authenticatedFetch module so tests don't make real HTTP requests.
vi.mock("@/lib/api-client", () => ({
  authenticatedFetch: vi.fn(),
}));

// Re-import after mock registration to get the mocked version.
import { authenticatedFetch } from "@/lib/api-client";

const mockFetch = authenticatedFetch as ReturnType<typeof vi.fn>;

function makeResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
    text: () => Promise.resolve(typeof body === "string" ? body : JSON.stringify(body)),
    blob: () => Promise.resolve(new Blob([JSON.stringify(body)])),
  } as unknown as Response;
}

const sampleJob = {
  id: "sbom-123",
  image_ref: "nginx:latest",
  host: "local",
  format: "spdx-json",
  status: "pending",
  created_at: 1700000000,
};

describe("getSBOMDownloadURL", () => {
  it("constructs the correct download URL", () => {
    const url = getSBOMDownloadURL("abc-456");
    expect(url).toContain("/api/v1/scan/sbom/abc-456");
    expect(url).toContain("download=true");
  });

  it("includes the job ID in the path", () => {
    const url = getSBOMDownloadURL("unique-job-id");
    expect(url).toMatch(/unique-job-id/);
  });
});

describe("generateSBOM", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("posts to the SBOM endpoint and returns the job", async () => {
    mockFetch.mockResolvedValueOnce(makeResponse(202, { job: sampleJob }));

    const result = await generateSBOM({
      imageRef: "nginx:latest",
      host: "local",
      format: "spdx-json",
    });

    expect(result).toEqual(sampleJob);
    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain("/api/v1/scan/sbom");
    expect(opts?.method).toBe("POST");
  });

  it("throws on non-ok response with server message", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      text: () => Promise.resolve("internal error"),
    } as unknown as Response);

    await expect(
      generateSBOM({ imageRef: "nginx:latest", host: "local" })
    ).rejects.toThrow("internal error");
  });

  it("throws with status code when body is empty on error", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 503,
      text: () => Promise.resolve(""),
    } as unknown as Response);

    await expect(
      generateSBOM({ imageRef: "nginx:latest", host: "local" })
    ).rejects.toThrow("503");
  });

  it("sends format in request body", async () => {
    mockFetch.mockResolvedValueOnce(makeResponse(202, { job: sampleJob }));

    await generateSBOM({ imageRef: "img:tag", host: "local", format: "cyclonedx-json" });

    const [, opts] = mockFetch.mock.calls[0];
    const body = JSON.parse(opts?.body as string);
    expect(body.format).toBe("cyclonedx-json");
  });
});

describe("getSBOMJob", () => {
  afterEach(() => vi.clearAllMocks());

  it("fetches the job by ID and returns it", async () => {
    const completedJob = { ...sampleJob, status: "complete" };
    mockFetch.mockResolvedValueOnce(makeResponse(200, { job: completedJob }));

    const result = await getSBOMJob("sbom-123");

    expect(result.status).toBe("complete");
    expect(mockFetch).toHaveBeenCalledOnce();
    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain("sbom-123");
  });

  it("throws on non-ok response", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 404,
      text: () => Promise.resolve("not found"),
    } as unknown as Response);

    await expect(getSBOMJob("missing-id")).rejects.toThrow("not found");
  });
});

describe("downloadSBOM", () => {
  afterEach(() => vi.clearAllMocks());

  it("fetches the download URL and returns a Blob", async () => {
    const blob = new Blob(["{}"], { type: "application/json" });
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      blob: () => Promise.resolve(blob),
    } as unknown as Response);

    const result = await downloadSBOM("sbom-123");

    expect(result).toBeInstanceOf(Blob);
    expect(mockFetch).toHaveBeenCalledOnce();
    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain("download=true");
  });

  it("throws on non-ok response", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 404,
      text: () => Promise.resolve("not found"),
    } as unknown as Response);

    await expect(downloadSBOM("bad-id")).rejects.toThrow("not found");
  });
});