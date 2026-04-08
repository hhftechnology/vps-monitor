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
import {
  getScanHistory,
  getScanHistoryDetail,
  getScannedImages,
  getAutoScanStatus,
  deleteScanHistory,
} from "../api/get-scan-history";
import {
  getSBOMHistory,
  getSBOMHistoryDetail,
  getSBOMedImages,
  deleteSBOMHistory,
} from "../api/get-sbom-history";
import type {
  ScannerConfig,
  HistoryQueryParams,
  SBOMHistoryQueryParams,
} from "../types";

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
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (params: GenerateSBOMParams) => generateSBOM(params),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sbomedImages"] });
      queryClient.invalidateQueries({ queryKey: ["sbomHistory"] });
    },
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

// --- History hooks ---

export function useScanHistory(params: HistoryQueryParams) {
  return useQuery({
    queryKey: ["scanHistory", params],
    queryFn: () => getScanHistory(params),
    staleTime: 10_000,
  });
}

export function useScanHistoryDetail(id: string | null) {
  return useQuery({
    queryKey: ["scanHistoryDetail", id],
    queryFn: () => getScanHistoryDetail(id!),
    enabled: !!id,
    staleTime: 60_000,
  });
}

export function useScannedImages() {
  return useQuery({
    queryKey: ["scannedImages"],
    queryFn: getScannedImages,
    staleTime: 30_000,
  });
}

export function useAutoScanStatus() {
  return useQuery({
    queryKey: ["autoScanStatus"],
    queryFn: getAutoScanStatus,
    staleTime: 15_000,
  });
}

export function useDeleteScanHistory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteScanHistory(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["scanHistory"] });
      queryClient.invalidateQueries({ queryKey: ["scannedImages"] });
    },
  });
}

export function useSBOMHistory(params: SBOMHistoryQueryParams) {
  return useQuery({
    queryKey: ["sbomHistory", params],
    queryFn: () => getSBOMHistory(params),
    staleTime: 10_000,
  });
}

export function useSBOMHistoryDetail(id: string | null) {
  return useQuery({
    queryKey: ["sbomHistoryDetail", id],
    queryFn: () => getSBOMHistoryDetail(id!),
    enabled: !!id,
    staleTime: 60_000,
  });
}

export function useSBOMedImages() {
  return useQuery({
    queryKey: ["sbomedImages"],
    queryFn: getSBOMedImages,
    staleTime: 30_000,
  });
}

export function useDeleteSBOMHistory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteSBOMHistory(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sbomHistory"] });
      queryClient.invalidateQueries({ queryKey: ["sbomedImages"] });
    },
  });
}
