import { useState } from "react";
import { format } from "date-fns";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  DownloadIcon,
  Trash2Icon,
  HistoryIcon,
  SearchIcon,
  XIcon,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { useScanHistory, useScanHistoryDetail, useScannedImages, useDeleteScanHistory } from "../hooks/use-scan-query";
import { exportScanHistory } from "../api/get-scan-history";
import { ScanResultsSummary } from "./scan-results-summary";
import { ScanResultsTable } from "./scan-results-table";
import type { HistoryQueryParams } from "../types";

export const SEVERITY_OPTIONS = ["Critical", "High", "Medium", "Low", "Negligible", "Unknown"] as const;
export type SeverityOption = typeof SEVERITY_OPTIONS[number] | "all" | "";

export function ScanHistoryPage() {
  const [params, setParams] = useState<HistoryQueryParams>({
    page: 1,
    page_size: 20,
    sort_by: "completed_at",
    sort_dir: "desc",
  });
  const [imageFilter, setImageFilter] = useState("");
  const [hostFilter, setHostFilter] = useState<string>("");
  const [severityFilter, setSeverityFilter] = useState<SeverityOption>("");
  const [selectedScanId, setSelectedScanId] = useState<string | null>(null);

  const { data: historyData, isLoading } = useScanHistory({
    ...params,
    image: imageFilter || undefined,
    host: hostFilter || undefined,
    min_severity: (severityFilter === "all" || severityFilter === "") ? undefined : severityFilter,
  });
  const { data: scannedImages } = useScannedImages();
  const { data: detailResult, isLoading: isDetailLoading } = useScanHistoryDetail(selectedScanId);
  const deleteMutation = useDeleteScanHistory();

  const uniqueHosts = Array.from(
    new Set(scannedImages?.map((img) => img.host) ?? [])
  );

  const clearFilters = () => {
    setImageFilter("");
    setHostFilter("");
    setSeverityFilter("");
    setParams((prev) => ({ ...prev, page: 1 }));
  };

  const hasFilters = imageFilter || hostFilter || severityFilter;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <HistoryIcon className="size-6" />
          <h1 className="text-2xl font-bold">Scan History</h1>
        </div>
        {historyData && (
          <p className="text-sm text-muted-foreground">
            {historyData.total} total scans
          </p>
        )}
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-[300px]">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            placeholder="Filter by image..."
            value={imageFilter}
            onChange={(e) => {
              setImageFilter(e.target.value);
              setParams((prev) => ({ ...prev, page: 1 }));
            }}
            className="pl-9"
          />
        </div>

        <Select value={hostFilter} onValueChange={(v) => {
          setHostFilter(v === "all" ? "" : v);
          setParams((prev) => ({ ...prev, page: 1 }));
        }}>
          <SelectTrigger className="w-[160px]">
            <SelectValue placeholder="All hosts" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All hosts</SelectItem>
            {uniqueHosts.map((host) => (
              <SelectItem key={host} value={host}>
                {host}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={severityFilter} onValueChange={(v: SeverityOption) => {
          setSeverityFilter(v === "all" ? "" : v);
          setParams((prev) => ({ ...prev, page: 1 }));
        }}>
          <SelectTrigger className="w-[160px]">
            <SelectValue placeholder="Any severity" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Any severity</SelectItem>
            <SelectItem value="Critical">Critical</SelectItem>
            <SelectItem value="High">High+</SelectItem>
            <SelectItem value="Medium">Medium+</SelectItem>
            <SelectItem value="Low">Low+</SelectItem>
          </SelectContent>
        </Select>

        {hasFilters && (
          <Button variant="ghost" size="sm" onClick={clearFilters}>
            <XIcon className="size-4 mr-1" />
            Clear
          </Button>
        )}
      </div>

      {/* Results Table */}
      <div className="border rounded-lg">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Image</TableHead>
              <TableHead>Host</TableHead>
              <TableHead>Scanner</TableHead>
              <TableHead>Vulnerabilities</TableHead>
              <TableHead
                className="cursor-pointer select-none"
                onClick={() =>
                  setParams((prev) => ({
                    ...prev,
                    sort_by: "completed_at",
                    sort_dir: prev.sort_dir === "desc" ? "asc" : "desc",
                  }))
                }
              >
                Date {params.sort_by === "completed_at" && (params.sort_dir === "desc" ? "\u2193" : "\u2191")}
              </TableHead>
              <TableHead>Duration</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                  Loading scan history...
                </TableCell>
              </TableRow>
            ) : !historyData?.results.length ? (
              <TableRow>
                <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                  No scan history found
                </TableCell>
              </TableRow>
            ) : (
              historyData.results.map((result) => (
                <TableRow
                  key={result.id}
                  className="cursor-pointer hover:bg-muted/50"
                  onClick={() => setSelectedScanId(result.id)}
                >
                  <TableCell className="font-mono text-sm max-w-[250px] truncate">
                    {result.image_ref}
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{result.host}</Badge>
                  </TableCell>
                  <TableCell className="capitalize">{result.scanner}</TableCell>
                  <TableCell>
                    <ScanResultsSummary summary={result.summary} />
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {format(new Date(result.completed_at * 1000), "MMM d, yyyy HH:mm")}
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {(result.duration_ms / 1000).toFixed(1)}s
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-2">
                      <Button
                        variant="ghost"
                        size="icon"
                        title="Export CSV"
                        onClick={(e) => {
                          e.stopPropagation();
                          exportScanHistory(result.id).catch((err) => {
                            console.error("Failed to export:", err);
                          });
                        }}
                      >
                        <DownloadIcon className="size-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        title="Delete"
                        disabled={deleteMutation.isPending}
                        onClick={(e) => {
                          e.stopPropagation();
                          if (confirm("Are you sure you want to delete this scan result?")) {
                            deleteMutation.mutate(result.id);
                          }
                        }}
                      >
                        <Trash2Icon className="size-4 text-destructive" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Pagination */}
      {historyData && historyData.total_pages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Page {historyData.page} of {historyData.total_pages}
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={historyData.page <= 1}
              onClick={() => setParams((prev) => ({ ...prev, page: (prev.page ?? 1) - 1 }))}
            >
              <ChevronLeftIcon className="size-4" />
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={historyData.page >= historyData.total_pages}
              onClick={() => setParams((prev) => ({ ...prev, page: (prev.page ?? 1) + 1 }))}
            >
              Next
              <ChevronRightIcon className="size-4" />
            </Button>
          </div>
        </div>
      )}

      {/* Detail Dialog */}
      <Dialog open={!!selectedScanId} onOpenChange={(open) => !open && setSelectedScanId(null)}>
        <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {detailResult ? (
                <span className="flex items-center gap-2">
                  Scan: <code className="text-sm">{detailResult.image_ref}</code>
                  <Badge variant="outline">{detailResult.host}</Badge>
                </span>
              ) : (
                "Scan Details"
              )}
            </DialogTitle>
          </DialogHeader>

          {isDetailLoading ? (
            <div className="py-8 text-center text-muted-foreground">Loading scan details...</div>
          ) : detailResult ? (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-muted-foreground">Scanner:</span>{" "}
                  <span className="capitalize">{detailResult.scanner}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Duration:</span>{" "}
                  {(detailResult.duration_ms / 1000).toFixed(1)}s
                </div>
                <div>
                  <span className="text-muted-foreground">Completed:</span>{" "}
                  {format(new Date(detailResult.completed_at * 1000), "MMM d, yyyy HH:mm:ss")}
                </div>
                <div>
                  <span className="text-muted-foreground">Total vulnerabilities:</span>{" "}
                  {detailResult.summary.total}
                </div>
              </div>

              <ScanResultsSummary summary={detailResult.summary} />

              {detailResult.vulnerabilities && detailResult.vulnerabilities.length > 0 && (
                <ScanResultsTable vulnerabilities={detailResult.vulnerabilities} />
              )}
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
