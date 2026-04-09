import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SBOMHistoryPage } from "./sbom-history-page";

const mockUseSBOMHistory = vi.fn();
const mockUseSBOMHistoryDetail = vi.fn();
const mockUseSBOMedImages = vi.fn();
const mockUseDeleteSBOMHistory = vi.fn();
const mockDownloadSBOMHistoryFile = vi.fn();

vi.mock("../hooks/use-scan-query", () => ({
  useSBOMHistory: (...args: unknown[]) => mockUseSBOMHistory(...args),
  useSBOMHistoryDetail: (...args: unknown[]) => mockUseSBOMHistoryDetail(...args),
  useSBOMedImages: (...args: unknown[]) => mockUseSBOMedImages(...args),
  useDeleteSBOMHistory: (...args: unknown[]) => mockUseDeleteSBOMHistory(...args),
}));

vi.mock("../api/get-sbom-history", () => ({
  downloadSBOMHistoryFile: (...args: unknown[]) => mockDownloadSBOMHistoryFile(...args),
}));

const historyPage = {
  results: [
    {
      id: "sbom-1",
      image_ref: "alpine:3.18",
      host: "local",
      format: "spdx-json",
      component_count: 5,
      file_size: 512,
      started_at: 100,
      completed_at: 200,
      duration_ms: 5000,
      components: [],
    },
  ],
  total: 1,
  page: 1,
  page_size: 20,
  total_pages: 1,
};

const detailResult = {
  ...historyPage.results[0],
  components: [
    {
      name: "busybox",
      version: "1.0.0",
      type: "package",
      purl: "pkg:apk/alpine/busybox@1.0.0",
    },
  ],
};

describe("SBOMHistoryPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseSBOMHistory.mockReturnValue({ data: historyPage, isLoading: false });
    mockUseSBOMHistoryDetail.mockImplementation((id: string | null) => ({
      data: id ? detailResult : null,
      isLoading: false,
    }));
    mockUseSBOMedImages.mockReturnValue({
      data: [{ image_ref: "alpine:3.18", host: "local", sbom_count: 1, last_sbom_at: 200 }],
    });
    mockUseDeleteSBOMHistory.mockReturnValue({ isPending: false, mutate: vi.fn() });
    mockDownloadSBOMHistoryFile.mockResolvedValue({
      blob: new Blob(["{}"], { type: "application/json" }),
      filename: "sbom-1.json",
    });
  });

  it("renders accessible sort controls with aria-sort state", () => {
    render(<SBOMHistoryPage />);

    expect(screen.getByRole("columnheader", { name: /Date/i })).toHaveAttribute("aria-sort", "descending");
    expect(screen.getByRole("columnheader", { name: /Components/i })).toHaveAttribute("aria-sort", "none");

    fireEvent.click(screen.getByRole("button", { name: /Components/i }));

    expect(screen.getByRole("columnheader", { name: /Components/i })).toHaveAttribute("aria-sort", "ascending");
    expect(screen.getByRole("columnheader", { name: /Date/i })).toHaveAttribute("aria-sort", "none");
  });

  it("opens the SBOM details dialog from the row button", () => {
    render(<SBOMHistoryPage />);

    fireEvent.click(screen.getByRole("button", { name: "alpine:3.18" }));

    expect(screen.getByText("busybox")).toBeInTheDocument();
    expect(screen.getByText(/SBOM:/)).toBeInTheDocument();
  });
});
