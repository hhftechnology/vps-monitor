import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { useObservedSBOMJobs } from "./use-scan-query";

const mockGetSBOMJob = vi.fn();
const mockToastSuccess = vi.fn();

vi.mock("../api/generate-sbom", () => ({
  generateSBOM: vi.fn(),
  getSBOMJob: (...args: unknown[]) => mockGetSBOMJob(...args),
}));

vi.mock("sonner", () => ({
  toast: {
    success: (...args: unknown[]) => mockToastSuccess(...args),
  },
}));

function ObserverHarness({
  jobIds,
  onTerminalJob,
}: {
  jobIds: string[];
  onTerminalJob: (jobId: string) => void;
}) {
  useObservedSBOMJobs(jobIds, onTerminalJob);
  return null;
}

describe("useObservedSBOMJobs", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("fires completion side effects from the parent observer", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    const onTerminalJob = vi.fn();

    mockGetSBOMJob.mockResolvedValue({
      id: "sbom-job-1",
      image_ref: "alpine:3.18",
      host: "local",
      format: "spdx-json",
      status: "complete",
      created_at: 200,
    });

    render(
      <QueryClientProvider client={queryClient}>
        <ObserverHarness jobIds={["sbom-job-1"]} onTerminalJob={onTerminalJob} />
      </QueryClientProvider>
    );

    await waitFor(() => {
      expect(onTerminalJob).toHaveBeenCalledWith("sbom-job-1");
    });

    expect(mockToastSuccess).toHaveBeenCalledWith("SBOM generated and saved to history");
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["sbomedImages"] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["sbomHistory"] });
  });
});
