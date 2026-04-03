import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { startScan, type StartScanParams } from "../api/start-scan";
import { startBulkScan, type StartBulkScanParams } from "../api/start-bulk-scan";
import { getScanJob, cancelScanJob } from "../api/get-scan-jobs";
import { getScanResults } from "../api/get-scan-results";
import { generateSBOM, getSBOMJob, type GenerateSBOMParams } from "../api/generate-sbom";
import {
  getScannerConfig,
  updateScannerConfig,
  testScanNotification,
} from "../api/scanner-config";
import type { ScannerConfig } from "../types";

const SCANNER_CONFIG_KEY = ["scannerConfig"] as const;

export function useScannerConfig() {
  return useQuery({
    queryKey: SCANNER_CONFIG_KEY,
    queryFn: getScannerConfig,
    staleTime: 30_000,
  });
}

export function useUpdateScannerConfig() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (config: ScannerConfig) => updateScannerConfig(config),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: SCANNER_CONFIG_KEY });
    },
  });
}

export function useTestScanNotification() {
  return useMutation({
    mutationFn: () => testScanNotification(),
  });
}

export function useStartScan() {
  return useMutation({
    mutationFn: (params: StartScanParams) => startScan(params),
  });
}

export function useStartBulkScan() {
  return useMutation({
    mutationFn: (params: StartBulkScanParams) => startBulkScan(params),
  });
}

export function useScanJob(id: string | null, enabled = true) {
  return useQuery({
    queryKey: ["scanJob", id],
    queryFn: () => getScanJob(id!),
    enabled: enabled && !!id,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (!data) return 2000;
      const job = data.job || data.bulkJob;
      if (!job) return false;
      const status = job.status;
      if (status === "complete" || status === "failed" || status === "cancelled") {
        return false;
      }
      return 2000;
    },
  });
}

export function useCancelScan() {
  return useMutation({
    mutationFn: (id: string) => cancelScanJob(id),
  });
}

export function useScanResults(imageRef: string, host: string, enabled = true) {
  return useQuery({
    queryKey: ["scanResults", imageRef, host],
    queryFn: () => getScanResults(imageRef, host),
    enabled,
    staleTime: 30_000,
  });
}

export function useGenerateSBOM() {
  return useMutation({
    mutationFn: (params: GenerateSBOMParams) => generateSBOM(params),
  });
}

export function useSBOMJob(id: string | null, enabled = true) {
  return useQuery({
    queryKey: ["sbomJob", id],
    queryFn: () => getSBOMJob(id!),
    enabled: enabled && !!id,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (!data) return 2000;
      if (data.status === "complete" || data.status === "failed" || data.status === "cancelled") {
        return false;
      }
      return 2000;
    },
  });
}
