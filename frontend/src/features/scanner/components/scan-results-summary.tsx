import { Badge } from "@/components/ui/badge";

import type { SeveritySummary } from "../types";

interface ScanResultsSummaryProps {
  summary: SeveritySummary;
}

const severityColors: Record<string, string> = {
  critical: "bg-red-600 text-white hover:bg-red-600",
  high: "bg-red-500 text-white hover:bg-red-500",
  medium: "bg-orange-500 text-white hover:bg-orange-500",
  low: "bg-yellow-500 text-white hover:bg-yellow-500",
};

export function ScanResultsSummary({ summary }: ScanResultsSummaryProps) {
  return (
    <div className="flex items-center gap-2 flex-wrap">
      {summary.critical > 0 && (
        <Badge className={severityColors.critical}>
          {summary.critical} Critical
        </Badge>
      )}
      {summary.high > 0 && (
        <Badge className={severityColors.high}>
          {summary.high} High
        </Badge>
      )}
      {summary.medium > 0 && (
        <Badge className={severityColors.medium}>
          {summary.medium} Medium
        </Badge>
      )}
      {summary.low > 0 && (
        <Badge className={severityColors.low}>
          {summary.low} Low
        </Badge>
      )}
      {summary.total === 0 && (
        <Badge variant="outline" className="border-green-500 text-green-500">
          No vulnerabilities
        </Badge>
      )}
    </div>
  );
}
