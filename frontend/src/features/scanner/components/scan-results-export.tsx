import { DownloadIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

import type { ScanResult } from "../types";

interface ScanResultsExportProps {
  result: ScanResult;
}

export function ScanResultsExport({ result }: ScanResultsExportProps) {
  const downloadFile = (content: string, filename: string, mimeType: string) => {
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  const exportJSON = () => {
    downloadFile(
      JSON.stringify(result, null, 2),
      `scan-${result.image_ref.replace(/[/:]/g, "_")}.json`,
      "application/json"
    );
  };

  const exportCSV = () => {
    const headers = ["CVE ID", "Severity", "Package", "Installed Version", "Fixed Version"];
    const rows = result.vulnerabilities.map((v) => [
      v.id,
      v.severity,
      v.package,
      v.installed_version,
      v.fixed_version || "",
    ]);

    const csv = [headers, ...rows].map((row) => row.map((cell) => `"${String(cell ?? "").replace(/"/g, '""')}"`).join(",")).join("\n");
    downloadFile(csv, `scan-${result.image_ref.replace(/[/:]/g, "_")}.csv`, "text/csv");
  };

  const exportMarkdown = () => {
    const lines = [
      `# Vulnerability Scan Report`,
      ``,
      `**Image:** ${result.image_ref}`,
      `**Host:** ${result.host}`,
      `**Scanner:** ${result.scanner}`,
      `**Duration:** ${(result.duration_ms / 1000).toFixed(1)}s`,
      `**Date:** ${new Date(result.completed_at * 1000).toLocaleString()}`,
      ``,
      `## Summary`,
      ``,
      `| Severity | Count |`,
      `|----------|-------|`,
      `| Critical | ${result.summary.critical} |`,
      `| High | ${result.summary.high} |`,
      `| Medium | ${result.summary.medium} |`,
      `| Low | ${result.summary.low} |`,
      `| **Total** | **${result.summary.total}** |`,
      ``,
      `## Vulnerabilities`,
      ``,
      `| CVE ID | Severity | Package | Installed | Fixed In |`,
      `|--------|----------|---------|-----------|----------|`,
      ...result.vulnerabilities.map((v) => {
        const ep = (s: string) => s.replace(/\|/g, "\\|");
        return `| ${ep(v.id)} | ${ep(v.severity)} | ${ep(v.package)} | ${ep(v.installed_version)} | ${ep(v.fixed_version || "-")} |`;
      }),
    ];

    downloadFile(
      lines.join("\n"),
      `scan-${result.image_ref.replace(/[/:]/g, "_")}.md`,
      "text/markdown"
    );
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          <DownloadIcon className="mr-2 size-4" />
          Export
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={exportMarkdown}>
          Markdown report (.md)
        </DropdownMenuItem>
        <DropdownMenuItem onClick={exportCSV}>
          CSV spreadsheet (.csv)
        </DropdownMenuItem>
        <DropdownMenuItem onClick={exportJSON}>
          JSON data (.json)
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
