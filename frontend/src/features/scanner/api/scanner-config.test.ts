import { afterEach, describe, expect, it, vi } from "vitest";

import type { ScannerConfig } from "../types";
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

const sampleConfig: ScannerConfig = {
  grypeImage: "anchore/grype:v0.110.0",
  trivyImage: "aquasec/trivy:0.69.3",
  syftImage: "anchore/syft:v1.27.1",
  defaultScanner: "grype",
  grypeArgs: "",
  trivyArgs: "",
  notifications: {
    onScanComplete: true,
    onBulkComplete: true,
    onNewCVEs: true,
    minSeverity: "High",
  },
  autoScan: { enabled: false, pollIntervalMinutes: 15 },
  forceRescan: false,
  scanTimeoutMinutes: 20,
  bulkTimeoutMinutes: 120,
  scannerMemoryMB: 2048,
  scannerPidsLimit: 512,
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
    const updatedConfig: ScannerConfig = { ...sampleConfig, defaultScanner: "trivy" };
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

describe("updateScannerConfig – resource limit fields", () => {
  afterEach(() => vi.clearAllMocks());

  it("includes scanTimeoutMinutes in the PUT body", async () => {
    const cfg = { ...sampleConfig, scanTimeoutMinutes: 45 };
    mockFetch.mockResolvedValueOnce(okResponse({ config: cfg }));

    await updateScannerConfig(cfg);

    const [, opts] = mockFetch.mock.calls[0];
    const body = JSON.parse(opts?.body as string);
    expect(body.scanTimeoutMinutes).toBe(45);
  });

  it("includes bulkTimeoutMinutes in the PUT body", async () => {
    const cfg = { ...sampleConfig, bulkTimeoutMinutes: 240 };
    mockFetch.mockResolvedValueOnce(okResponse({ config: cfg }));

    await updateScannerConfig(cfg);

    const [, opts] = mockFetch.mock.calls[0];
    const body = JSON.parse(opts?.body as string);
    expect(body.bulkTimeoutMinutes).toBe(240);
  });

  it("includes scannerMemoryMB in the PUT body", async () => {
    const cfg = { ...sampleConfig, scannerMemoryMB: 4096 };
    mockFetch.mockResolvedValueOnce(okResponse({ config: cfg }));

    await updateScannerConfig(cfg);

    const [, opts] = mockFetch.mock.calls[0];
    const body = JSON.parse(opts?.body as string);
    expect(body.scannerMemoryMB).toBe(4096);
  });

  it("includes scannerPidsLimit in the PUT body", async () => {
    const cfg = { ...sampleConfig, scannerPidsLimit: 1024 };
    mockFetch.mockResolvedValueOnce(okResponse({ config: cfg }));

    await updateScannerConfig(cfg);

    const [, opts] = mockFetch.mock.calls[0];
    const body = JSON.parse(opts?.body as string);
    expect(body.scannerPidsLimit).toBe(1024);
  });

  it("returns the resource limit values echoed back by the server", async () => {
    const cfg = {
      ...sampleConfig,
      scanTimeoutMinutes: 30,
      bulkTimeoutMinutes: 90,
      scannerMemoryMB: 1024,
      scannerPidsLimit: 256,
    };
    mockFetch.mockResolvedValueOnce(okResponse({ config: cfg }));

    const result = await updateScannerConfig(cfg);

    expect(result.scanTimeoutMinutes).toBe(30);
    expect(result.bulkTimeoutMinutes).toBe(90);
    expect(result.scannerMemoryMB).toBe(1024);
    expect(result.scannerPidsLimit).toBe(256);
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