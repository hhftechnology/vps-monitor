import { fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { ScanResult } from "../types";
import { ScanResultsExport } from "./scan-results-export";

const createObjectURL = vi.fn(() => "blob:test");
const revokeObjectURL = vi.fn();
Object.defineProperty(URL, "createObjectURL", { value: createObjectURL, writable: true });
Object.defineProperty(URL, "revokeObjectURL", { value: revokeObjectURL, writable: true });

const sampleResult: ScanResult = {
  id: "result-1",
  image_ref: "nginx:latest",
  host: "local",
  scanner: "grype",
  vulnerabilities: [
    {
      id: "CVE-2023-0001",
      severity: "High",
      package: "openssl",
      installed_version: "1.1.1t",
      fixed_version: "1.1.1u",
    },
    {
      id: "CVE-2023-0002",
      severity: "Medium",
      package: "curl",
      installed_version: "7.88.0",
    },
  ],
  summary: { critical: 0, high: 1, medium: 1, low: 0, negligible: 0, unknown: 0, total: 2 },
  started_at: 1700000000,
  completed_at: 1700000100,
  duration_ms: 100000,
};

function openMenu() {
  fireEvent.pointerDown(screen.getByRole("button", { name: /Export/i }), {
    button: 0,
    ctrlKey: false,
  });
}

describe("ScanResultsExport", () => {
  let anchorClickSpy: ReturnType<typeof vi.spyOn>;
  let anchorElement: HTMLAnchorElement;
  let originalCreateElement: typeof document.createElement;

  beforeEach(() => {
    originalCreateElement = document.createElement.bind(document);
    anchorElement = originalCreateElement("a");
    anchorClickSpy = vi.spyOn(anchorElement, "click").mockImplementation(() => {});
    vi.spyOn(document, "createElement").mockImplementation((tagName: string, options?: ElementCreationOptions) => {
      if (tagName === "a") {
        return anchorElement;
      }
      return originalCreateElement(tagName, options);
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.clearAllMocks();
  });

  it("renders the Export button", () => {
    render(<ScanResultsExport result={sampleResult} />);
    expect(screen.getByRole("button", { name: /Export/i })).toBeInTheDocument();
  });

  it("opens dropdown with format options when the trigger is clicked", () => {
    render(<ScanResultsExport result={sampleResult} />);
    openMenu();

    expect(screen.getByText("Markdown report (.md)")).toBeInTheDocument();
    expect(screen.getByText("CSV spreadsheet (.csv)")).toBeInTheDocument();
    expect(screen.getByText("JSON data (.json)")).toBeInTheDocument();
  });

  it("triggers a download with .json extension when JSON is selected", () => {
    render(<ScanResultsExport result={sampleResult} />);
    openMenu();
    fireEvent.click(screen.getByText("JSON data (.json)"));

    expect(createObjectURL).toHaveBeenCalled();
    expect(anchorClickSpy).toHaveBeenCalled();
    expect(anchorElement.download).toMatch(/\.json$/);
  });

  it("triggers a download with .csv extension when CSV is selected", () => {
    render(<ScanResultsExport result={sampleResult} />);
    openMenu();
    fireEvent.click(screen.getByText("CSV spreadsheet (.csv)"));

    expect(createObjectURL).toHaveBeenCalled();
    expect(anchorElement.download).toMatch(/\.csv$/);
  });

  it("triggers a download with .md extension when Markdown is selected", () => {
    render(<ScanResultsExport result={sampleResult} />);
    openMenu();
    fireEvent.click(screen.getByText("Markdown report (.md)"));

    expect(createObjectURL).toHaveBeenCalled();
    expect(anchorElement.download).toMatch(/\.md$/);
  });

  it("uses image_ref in the filename with special chars replaced", () => {
    render(<ScanResultsExport result={sampleResult} />);
    openMenu();
    fireEvent.click(screen.getByText("JSON data (.json)"));

    expect(anchorElement.download).toMatch(/nginx_latest/);
  });

  it("revokes the object URL after triggering the download", () => {
    render(<ScanResultsExport result={sampleResult} />);
    openMenu();
    fireEvent.click(screen.getByText("JSON data (.json)"));

    expect(revokeObjectURL).toHaveBeenCalledWith("blob:test");
  });
});
