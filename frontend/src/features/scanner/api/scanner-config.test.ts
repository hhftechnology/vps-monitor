import { afterEach, describe, expect, it, vi } from "vitest";

import { getScannerConfig, testScanNotification, updateScannerConfig } from "./scanner-config";

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

const sampleConfig = {
  grypeImage: "anchore/grype:v0.110.0",
  trivyImage: "aquasec/trivy:0.69.3",
  syftImage: "anchore/syft:v1.27.1",
  defaultScanner: "grype" as const,
  grypeArgs: "",
  trivyArgs: "",
  notifications: {
    onScanComplete: true,
    onBulkComplete: true,
    minSeverity: "High" as const,
  },
};

describe("getScannerConfig", () => {
  afterEach(() => vi.clearAllMocks());

  it("returns the scanner config", async () => {
    mockFetch.mockResolvedValueOnce(okResponse({ config: sampleConfig }));

    const result = await getScannerConfig();

    expect(result).toEqual(sampleConfig);
    expect(mockFetch).toHaveBeenCalledOnce();
    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain("/api/v1/settings/scan");
  });

  it("throws on server error", async () => {
    mockFetch.mockResolvedValueOnce(errResponse(500, "error"));

    await expect(getScannerConfig()).rejects.toThrow("error");
  });
});

describe("updateScannerConfig", () => {
  afterEach(() => vi.clearAllMocks());

  it("sends a PUT request with the config and returns updated config", async () => {
    const updatedConfig = { ...sampleConfig, defaultScanner: "trivy" as const };
    mockFetch.mockResolvedValueOnce(okResponse({ config: updatedConfig }));

    const result = await updateScannerConfig(updatedConfig);

    expect(result.defaultScanner).toBe("trivy");
    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain("/api/v1/settings/scan");
    expect(opts?.method).toBe("PUT");
    const body = JSON.parse(opts?.body as string);
    expect(body.defaultScanner).toBe("trivy");
  });

  it("throws on server error", async () => {
    mockFetch.mockResolvedValueOnce(errResponse(400, "invalid config"));

    await expect(updateScannerConfig(sampleConfig)).rejects.toThrow("invalid config");
  });
});

describe("testScanNotification", () => {
  afterEach(() => vi.clearAllMocks());

  it("sends a POST request to the test-notification endpoint", async () => {
    mockFetch.mockResolvedValueOnce({ ok: true, status: 200 } as Response);

    await testScanNotification();

    expect(mockFetch).toHaveBeenCalledOnce();
    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain("test-notification");
    expect(opts?.method).toBe("POST");
  });

  it("throws on non-ok response", async () => {
    mockFetch.mockResolvedValueOnce(errResponse(500, "notification failed"));

    await expect(testScanNotification()).rejects.toThrow("notification failed");
  });
});