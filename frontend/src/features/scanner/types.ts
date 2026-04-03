export type ScannerType = "grype" | "trivy";
export type SeverityLevel = "Critical" | "High" | "Medium" | "Low" | "Negligible" | "Unknown";
export type ScanJobStatus = "pending" | "pulling_scanner" | "scanning" | "complete" | "failed" | "cancelled";
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
  created_at: number;
  error?: string;
}

export interface NotificationConfig {
  discordWebhookURL?: string;
  slackWebhookURL?: string;
  onScanComplete: boolean;
  onBulkComplete: boolean;
  minSeverity?: SeverityLevel;
}

export interface ScannerConfig {
  grypeImage: string;
  trivyImage: string;
  syftImage: string;
  defaultScanner: ScannerType;
  grypeArgs: string;
  trivyArgs: string;
  notifications: NotificationConfig;
}
