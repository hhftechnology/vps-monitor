/**
 * Log Types for Docker Logs Viewer
 * Enhanced structured logs with parsed timestamps and levels
 */

// ============================================================================
// Log Entry Types
// ============================================================================

export interface LogEntry {
  /** Parsed timestamp in ISO8601 format */
  timestamp: string;
  /** Detected log level */
  level: LogLevel;
  /** Clean log message (ANSI codes removed) */
  message: string;
  /** Stream source: stdout or stderr */
  stream: "stdout" | "stderr";
  /** Original unchanged log line */
  raw: string;
}

export type LogLevel =
  | "TRACE"
  | "DEBUG"
  | "INFO"
  | "WARN"
  | "WARNING"
  | "ERROR"
  | "FATAL"
  | "PANIC"
  | "UNKNOWN";

// ============================================================================
// API Response Types
// ============================================================================

export interface ParsedLogsResponse {
  logs: LogEntry[];
  count: number;
}

// ============================================================================
// UI Helper Types
// ============================================================================

export type LogLevelFilter = LogLevel | "ALL";

export interface LogFilterOptions {
  level: LogLevelFilter;
  searchTerm: string;
  since?: string;
  until?: string;
  stream?: "stdout" | "stderr" | "both";
}

// ============================================================================
// Utility Functions
// ============================================================================

/** Get Tailwind color class for log level */
export function getLogLevelColor(level: LogLevel): string {
  switch (level) {
    case "TRACE":
    case "DEBUG":
      return "text-gray-500";
    case "INFO":
      return "text-blue-600";
    case "WARN":
    case "WARNING":
      return "text-yellow-600";
    case "ERROR":
      return "text-red-600";
    case "FATAL":
    case "PANIC":
      return "text-red-800";
    default:
      return "text-gray-700";
  }
}

/** Get background color class for log level */
export function getLogLevelBgColor(level: LogLevel): string {
  switch (level) {
    case "TRACE":
    case "DEBUG":
      return "bg-gray-100";
    case "INFO":
      return "bg-blue-50";
    case "WARN":
    case "WARNING":
      return "bg-yellow-50";
    case "ERROR":
      return "bg-red-50";
    case "FATAL":
    case "PANIC":
      return "bg-red-100";
    default:
      return "bg-gray-50";
  }
}

/** Get badge color for log level (for shadcn/ui Badge component) */
export function getLogLevelBadgeVariant(
  level: LogLevel
): "default" | "secondary" | "destructive" | "outline" {
  switch (level) {
    case "ERROR":
    case "FATAL":
    case "PANIC":
      return "destructive";
    case "WARN":
    case "WARNING":
      return "outline";
    case "INFO":
      return "default";
    default:
      return "secondary";
  }
}

/** Get priority number for log level (higher = more severe) */
export function getLogLevelPriority(level: LogLevel): number {
  switch (level) {
    case "TRACE":
      return 0;
    case "DEBUG":
      return 1;
    case "INFO":
      return 2;
    case "WARN":
    case "WARNING":
      return 3;
    case "ERROR":
      return 4;
    case "FATAL":
    case "PANIC":
      return 5;
    default:
      return 2;
  }
}

/** Format timestamp for display */
export function formatLogTimestamp(
  timestamp: string,
  format: "time" | "full" | "relative" = "time"
): string {
  const date = new Date(timestamp);

  switch (format) {
    case "time":
      return date.toLocaleTimeString();
    case "full":
      return date.toLocaleString();
    case "relative":
      return getRelativeTime(date);
    default:
      return timestamp;
  }
}

/** Get relative time (e.g., "2 minutes ago") */
function getRelativeTime(date: Date): string {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffSec < 60) return `${diffSec}s ago`;
  if (diffMin < 60) return `${diffMin}m ago`;
  if (diffHour < 24) return `${diffHour}h ago`;
  return `${diffDay}d ago`;
}

/** Filter logs by multiple criteria */
export function filterLogs(
  logs: LogEntry[],
  options: LogFilterOptions
): LogEntry[] {
  let filtered = logs;

  // Filter by level
  if (options.level !== "ALL") {
    filtered = filtered.filter((log) => log.level === options.level);
  }

  // Filter by search term
  if (options.searchTerm) {
    const searchLower = options.searchTerm.toLowerCase();
    filtered = filtered.filter((log) =>
      log.message.toLowerCase().includes(searchLower)
    );
  }

  // Filter by stream
  if (options.stream && options.stream !== "both") {
    filtered = filtered.filter((log) => log.stream === options.stream);
  }

  return filtered;
}

/** Count logs by level */
export function countLogsByLevel(logs: LogEntry[]): Record<LogLevel, number> {
  const counts: Record<LogLevel, number> = {
    TRACE: 0,
    DEBUG: 0,
    INFO: 0,
    WARN: 0,
    WARNING: 0,
    ERROR: 0,
    FATAL: 0,
    PANIC: 0,
    UNKNOWN: 0,
  };

  for (const log of logs) {
    counts[log.level]++;
  }

  return counts;
}

/** Get log level options for dropdown */
export const LOG_LEVEL_OPTIONS: Array<{
  value: LogLevelFilter;
  label: string;
}> = [
  { value: "ALL", label: "All Levels" },
  { value: "ERROR", label: "Errors" },
  { value: "WARN", label: "Warnings" },
  { value: "INFO", label: "Info" },
  { value: "DEBUG", label: "Debug" },
  { value: "TRACE", label: "Trace" },
];

/** Check if log is an error level */
export function isErrorLog(log: LogEntry): boolean {
  return (
    log.level === "ERROR" || log.level === "FATAL" || log.level === "PANIC"
  );
}

/** Check if log is a warning level */
export function isWarningLog(log: LogEntry): boolean {
  return log.level === "WARN" || log.level === "WARNING";
}

/** Highlight search term in message */
export function highlightSearchTerm(
  message: string,
  searchTerm: string
): string {
  if (!searchTerm) return message;

  const regex = new RegExp(`(${searchTerm})`, "gi");
  return message.replace(regex, "<mark>$1</mark>");
}
