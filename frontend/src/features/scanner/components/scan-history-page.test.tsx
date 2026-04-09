import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ScanHistoryPage } from "./scan-history-page";

const mockUseScanHistory = vi.fn();
const mockUseScanHistoryDetail = vi.fn();
const mockUseScannedImages = vi.fn();
const mockUseDeleteScanHistory = vi.fn();
const mockExportScanHistory = vi.fn();

vi.mock("../hooks/use-scan-query", () => ({
  useScanHistory: (...args: unknown[]) => mockUseScanHistory(...args),
  useScanHistoryDetail: (...args: unknown[]) => mockUseScanHistoryDetail(...args),
  useScannedImages: (...args: unknown[]) => mockUseScannedImages(...args),
  useDeleteScanHistory: (...args: unknown[]) => mockUseDeleteScanHistory(...args),
}));

vi.mock("../api/get-scan-history", () => ({
  exportScanHistory: (...args: unknown[]) => mockExportScanHistory(...args),
}));

vi.mock("./scan-results-summary", () => ({
  ScanResultsSummary: () => <div>summary</div>,
}));

vi.mock("./scan-results-table", () => ({
  ScanResultsTable: () => <div>table</div>,
}));

const historyPage = {
  results: [
    {
      id: "scan-1",
      image_ref: "redis:7",
      host: "local",
      scanner: "grype",
      summary: {
        critical: 0,
        high: 1,
        medium: 0,
        low: 0,
        negligible: 0,
        unknown: 0,
        total: 1,
      },
      vulnerabilities: [],
      started_at: 100,
      completed_at: 200,
      duration_ms: 5000,
    },
  ],
  total: 1,
  page: 1,
  page_size: 20,
  total_pages: 1,
};

describe("ScanHistoryPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseScanHistory.mockReturnValue({ data: historyPage, isLoading: false });
    mockUseScanHistoryDetail.mockImplementation((id: string | null) => ({
      data: id
        ? {
            ...historyPage.results[0],
            vulnerabilities: [
              {
                id: "CVE-2024-0001",
                severity: "High",
                package: "redis",
                installed_version: "1.0.0",
              },
            ],
          }
        : null,
      isLoading: false,
    }));
    mockUseScannedImages.mockReturnValue({
      data: [{ image_ref: "redis:7", host: "local", scan_count: 1, last_scanned: 200 }],
    });
    mockUseDeleteScanHistory.mockReturnValue({ isPending: false, mutate: vi.fn() });
    mockExportScanHistory.mockResolvedValue(undefined);
  });

  it("renders the date sort as an explicit button with aria-sort", () => {
    render(<ScanHistoryPage />);

    expect(screen.getByRole("columnheader", { name: /Date/i })).toHaveAttribute("aria-sort", "descending");
    expect(screen.getByRole("button", { name: /Date/i })).toBeInTheDocument();
  });

  it("opens the scan details dialog from the row button", () => {
    render(<ScanHistoryPage />);

    fireEvent.click(screen.getByRole("button", { name: "redis:7" }));

    expect(screen.getByText(/Scan:/)).toBeInTheDocument();
    expect(screen.getByText("table")).toBeInTheDocument();
  });
});
