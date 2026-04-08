export type ScannerType = "grype" | "trivy";
export type SeverityLevel = "Critical" | "High" | "Medium" | "Low" | "Negligible" | "Unknown";
export type ScanJobStatus = "pending" | "pulling_scanner" | "scanning" | "complete" | "failed" | "cancelled" | "expired";
export type SBOMFormat = "spdx-json" | "cyclonedx-json";

export interface Vulnerability {
  id: string;
  severity: SeverityLevel;
  package: string;
  installed_version: string;
  fixed_version?: string;
  description?: string;
  data_source?: string;
}

export interface SeveritySummary {
  critical: number;
  high: number;
  medium: number;
  low: number;
  negligible: number;
  unknown: number;
  total: number;
}

export interface ScanResult {
  id: string;
  image_ref: string;
  host: string;
  scanner: ScannerType;
  vulnerabilities: Vulnerability[];
  summary: SeveritySummary;
  started_at: number;
  completed_at: number;
  duration_ms: number;
  error?: string;
}

export interface ScanJob {
  id: string;
  image_ref: string;
  host: string;
  scanner: ScannerType;
  status: ScanJobStatus;
  progress?: string;
  result?: ScanResult;
  created_at: number;
  error?: string;
}

export interface BulkScanJob {
  id: string;
  jobs: ScanJob[];
  total_images: number;
  completed: number;
  failed: number;
  status: ScanJobStatus;
  created_at: number;
}

export interface SBOMJob {
  id: string;
  image_ref: string;
  host: string;
  format: SBOMFormat;
  status: ScanJobStatus;
  result_id?: string;
  created_at: number;
  error?: string;
}

export interface SBOMComponent {
  name: string;
  version: string;
  type: string;
  purl: string;
}

export interface SBOMResult {
  id: string;
  image_ref: string;
  host: string;
  format: SBOMFormat;
  component_count: number;
  file_size: number;
  started_at: number;
  completed_at: number;
  duration_ms: number;
  error?: string;
  components?: SBOMComponent[];
}

export interface NotificationConfig {
  discordWebhookURL?: string;
  slackWebhookURL?: string;
  onScanComplete: boolean;
  onBulkComplete: boolean;
  onNewCVEs: boolean;
  minSeverity?: SeverityLevel;
}

export interface AutoScanConfig {
  enabled: boolean;
  pollIntervalMinutes?: number;
}

export interface ScannerConfig {
  grypeImage: string;
  trivyImage: string;
  syftImage: string;
  defaultScanner: ScannerType;
  grypeArgs: string;
  trivyArgs: string;
  notifications: NotificationConfig;
  autoScan: AutoScanConfig;
  forceRescan: boolean;
  scanTimeoutMinutes: number;
  bulkTimeoutMinutes: number;
  scannerMemoryMB: number;
  scannerPidsLimit: number;
}

export interface HistoryQueryParams {
  image?: string;
  host?: string;
  min_severity?: SeverityLevel;
  start_date?: number;
  end_date?: number;
  page?: number;
  page_size?: number;
  sort_by?: "completed_at" | "summary_total" | "summary_critical";
  sort_dir?: "asc" | "desc";
}

export interface HistoryPage {
  results: ScanResult[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface SBOMHistoryQueryParams {
  image?: string;
  host?: string;
  format?: SBOMFormat;
  start_date?: number;
  end_date?: number;
  page?: number;
  page_size?: number;
  sort_by?: "completed_at" | "component_count";
  sort_dir?: "asc" | "desc";
}

export interface SBOMHistoryPage {
  results: SBOMResult[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface ScannedImage {
  image_ref: string;
  host: string;
  scan_count: number;
  last_scanned: number;
}

export interface SBOMedImage {
  image_ref: string;
  host: string;
  sbom_count: number;
  last_sbom_at: number;
}

export interface AutoScanStatus {
  enabled: boolean;
  lastPollAt: number;
  eventsConnected: Record<string, boolean>;
}

export interface RescanBlockedResponse {
  error: "image_unchanged";
  message: string;
  last_scan_id?: string;
  last_scan_at?: number;
}

export interface SBOMRescanBlockedResponse {
  error: "image_unchanged";
  message: string;
  last_sbom_id?: string;
  last_sbom_at?: number;
}
