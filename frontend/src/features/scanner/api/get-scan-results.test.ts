import { afterEach, describe, expect, it, vi } from "vitest";

import { getLatestScanResult, getScanResults } from "./get-scan-results";

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

const sampleResult = {
  id: "result-1",
  image_ref: "nginx:latest",
  host: "local",
  scanner: "grype",
  vulnerabilities: [],
  summary: { critical: 0, high: 2, medium: 0, low: 0, negligible: 0, unknown: 0, total: 2 },
  started_at: 1700000000,
  completed_at: 1700000100,
  duration_ms: 100000,
};

describe("getScanResults", () => {
  afterEach(() => vi.clearAllMocks());

  it("returns an array of scan results", async () => {
    mockFetch.mockResolvedValueOnce(okResponse({ results: [sampleResult] }));

    const results = await getScanResults("nginx:latest", "local");

    expect(results).toHaveLength(1);
    expect(results[0]).toEqual(sampleResult);
  });

  it("URL-encodes imageRef in the request path", async () => {
    mockFetch.mockResolvedValueOnce(okResponse({ results: [] }));

    await getScanResults("my-registry.example.com:5000/myapp:v1.0", "local");

    const [url] = mockFetch.mock.calls[0];
    // The image ref with colon and slash must be encoded in the URL
    expect(url).toContain(encodeURIComponent("my-registry.example.com:5000/myapp:v1.0"));
  });

  it("includes host as a query parameter", async () => {
    mockFetch.mockResolvedValueOnce(okResponse({ results: [] }));

    await getScanResults("nginx:latest", "my-remote-host");

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain("host=my-remote-host");
  });

  it("throws on server error", async () => {
    mockFetch.mockResolvedValueOnce(errResponse(500, "server error"));

    await expect(getScanResults("nginx:latest", "local")).rejects.toThrow("server error");
  });
});

describe("getLatestScanResult", () => {
  afterEach(() => vi.clearAllMocks());

  it("returns the latest result", async () => {
    mockFetch.mockResolvedValueOnce(okResponse({ result: sampleResult }));

    const result = await getLatestScanResult("nginx:latest", "local");

    expect(result).toEqual(sampleResult);
    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain("/latest");
  });

  it("returns null on 404", async () => {
    mockFetch.mockResolvedValueOnce({ ok: false, status: 404 } as Response);

    const result = await getLatestScanResult("missing:image", "local");

    expect(result).toBeNull();
  });

  it("throws on non-404 errors", async () => {
    mockFetch.mockResolvedValueOnce(errResponse(500, "internal error"));

    await expect(getLatestScanResult("nginx:latest", "local")).rejects.toThrow("internal error");
  });

  it("encodes host in query string", async () => {
    mockFetch.mockResolvedValueOnce(okResponse({ result: sampleResult }));

    await getLatestScanResult("nginx:latest", "host with spaces");

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain(encodeURIComponent("host with spaces"));
  });
});