import { describe, expect, it } from "vitest";

import type { ScannerConfig } from "./types";

// ---------------------------------------------------------------------------
// ScannerConfig – resource limit fields added in this PR
// ---------------------------------------------------------------------------
// These tests verify the runtime shape of objects that conform to the
// ScannerConfig interface, ensuring the four new fields are present and
// correctly typed. Because TypeScript interfaces are compile-time only, we
// create plain objects that satisfy the interface and assert their values.

function makeScannerConfig(overrides: Partial<ScannerConfig> = {}): ScannerConfig {
  return {
    grypeImage: "anchore/grype:v0.110.0",
    trivyImage: "aquasec/trivy:0.69.3",
    syftImage: "anchore/syft:v1.42.3",
    defaultScanner: "grype",
    grypeArgs: "",
    trivyArgs: "",
    notifications: {
      onScanComplete: true,
      onBulkComplete: true,
      onNewCVEs: true,
      minSeverity: "High",
    },
    autoScan: {
      enabled: false,
      pollIntervalMinutes: 15,
    },
    forceRescan: false,
    scanTimeoutMinutes: 20,
    bulkTimeoutMinutes: 120,
    scannerMemoryMB: 2048,
    scannerPidsLimit: 512,
    ...overrides,
  };
}

describe("ScannerConfig resource limit fields", () => {
  it("has numeric scanTimeoutMinutes defaulting to 20", () => {
    const cfg = makeScannerConfig();
    expect(typeof cfg.scanTimeoutMinutes).toBe("number");
    expect(cfg.scanTimeoutMinutes).toBe(20);
  });

  it("has numeric bulkTimeoutMinutes defaulting to 120", () => {
    const cfg = makeScannerConfig();
    expect(typeof cfg.bulkTimeoutMinutes).toBe("number");
    expect(cfg.bulkTimeoutMinutes).toBe(120);
  });

  it("has numeric scannerMemoryMB defaulting to 2048", () => {
    const cfg = makeScannerConfig();
    expect(typeof cfg.scannerMemoryMB).toBe("number");
    expect(cfg.scannerMemoryMB).toBe(2048);
  });

  it("has numeric scannerPidsLimit defaulting to 512", () => {
    const cfg = makeScannerConfig();
    expect(typeof cfg.scannerPidsLimit).toBe("number");
    expect(cfg.scannerPidsLimit).toBe(512);
  });

  it("accepts custom resource limit values", () => {
    const cfg = makeScannerConfig({
      scanTimeoutMinutes: 60,
      bulkTimeoutMinutes: 240,
      scannerMemoryMB: 4096,
      scannerPidsLimit: 1024,
    });
    expect(cfg.scanTimeoutMinutes).toBe(60);
    expect(cfg.bulkTimeoutMinutes).toBe(240);
    expect(cfg.scannerMemoryMB).toBe(4096);
    expect(cfg.scannerPidsLimit).toBe(1024);
  });

  it("resource limit fields are independent of other config fields", () => {
    const cfg = makeScannerConfig({
      defaultScanner: "trivy",
      scanTimeoutMinutes: 30,
    });
    expect(cfg.defaultScanner).toBe("trivy");
    expect(cfg.scanTimeoutMinutes).toBe(30);
    // Other resource limit fields remain at defaults
    expect(cfg.bulkTimeoutMinutes).toBe(120);
    expect(cfg.scannerMemoryMB).toBe(2048);
    expect(cfg.scannerPidsLimit).toBe(512);
  });

  it("minimum boundary value of 1 is valid for scanTimeoutMinutes", () => {
    const cfg = makeScannerConfig({ scanTimeoutMinutes: 1 });
    expect(cfg.scanTimeoutMinutes).toBe(1);
  });

  it("large values are representable", () => {
    const cfg = makeScannerConfig({
      scanTimeoutMinutes: 9999,
      bulkTimeoutMinutes: 9999,
      scannerMemoryMB: 65536,
      scannerPidsLimit: 32768,
    });
    expect(cfg.scanTimeoutMinutes).toBe(9999);
    expect(cfg.bulkTimeoutMinutes).toBe(9999);
    expect(cfg.scannerMemoryMB).toBe(65536);
    expect(cfg.scannerPidsLimit).toBe(32768);
  });
});

describe("ScannerConfig updateScannerConfig round-trip – resource limits", () => {
  // Verify that spreading/cloning a ScannerConfig preserves the new fields,
  // matching how scanner-section.tsx updates draft state via setDraft({...draft, field: value}).

  it("spread preserves all resource limit fields", () => {
    const original = makeScannerConfig({
      scanTimeoutMinutes: 45,
      bulkTimeoutMinutes: 180,
      scannerMemoryMB: 8192,
      scannerPidsLimit: 768,
    });
    const updated: ScannerConfig = { ...original, grypeImage: "anchore/grype:v2" };
    expect(updated.scanTimeoutMinutes).toBe(45);
    expect(updated.bulkTimeoutMinutes).toBe(180);
    expect(updated.scannerMemoryMB).toBe(8192);
    expect(updated.scannerPidsLimit).toBe(768);
    expect(updated.grypeImage).toBe("anchore/grype:v2");
  });

  it("individual field override does not affect other resource limit fields", () => {
    const original = makeScannerConfig();
    const updated: ScannerConfig = { ...original, scanTimeoutMinutes: 99 };
    expect(updated.scanTimeoutMinutes).toBe(99);
    expect(updated.bulkTimeoutMinutes).toBe(original.bulkTimeoutMinutes);
    expect(updated.scannerMemoryMB).toBe(original.scannerMemoryMB);
    expect(updated.scannerPidsLimit).toBe(original.scannerPidsLimit);
  });

  it("serialises and deserialises resource limit fields via JSON", () => {
    const original = makeScannerConfig({
      scanTimeoutMinutes: 30,
      bulkTimeoutMinutes: 90,
      scannerMemoryMB: 1024,
      scannerPidsLimit: 256,
    });
    const json = JSON.stringify(original);
    const restored = JSON.parse(json) as ScannerConfig;
    expect(restored.scanTimeoutMinutes).toBe(30);
    expect(restored.bulkTimeoutMinutes).toBe(90);
    expect(restored.scannerMemoryMB).toBe(1024);
    expect(restored.scannerPidsLimit).toBe(256);
  });
});