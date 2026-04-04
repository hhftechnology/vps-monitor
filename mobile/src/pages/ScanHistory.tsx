import { useState } from "react";
import { format } from "date-fns";
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  History,
  Loader2,
  Search,
  X,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ScanResultPanel } from "@/features/scanner/components/ScanResultPanel";
import { useScanHistory, useScanHistoryDetail, useScannedImages } from "@/features/scanner/hooks";
import { getSeverityClass } from "@/features/scanner/utils";
import type { HistoryQueryParams, SeverityLevel } from "@/features/scanner/types";

export default function ScanHistory() {
  const [params, setParams] = useState<HistoryQueryParams>({
    page: 1,
    page_size: 20,
    sort_by: "completed_at",
    sort_dir: "desc",
  });
  const [imageFilter, setImageFilter] = useState("");
  const [hostFilter, setHostFilter] = useState("");
  const [severityFilter, setSeverityFilter] = useState("");
  const [selectedScanId, setSelectedScanId] = useState<string | null>(null);
  const [showFilters, setShowFilters] = useState(false);

  const { data: historyData, isLoading } = useScanHistory({
    ...params,
    image: imageFilter || undefined,
    host: hostFilter || undefined,
    min_severity: (severityFilter as SeverityLevel) || undefined,
  });
  const { data: scannedImages } = useScannedImages();
  const { data: detailResult, isLoading: isDetailLoading } = useScanHistoryDetail(selectedScanId);

  const uniqueHosts = Array.from(
    new Set(scannedImages?.map((img) => img.host) ?? [])
  );

  const hasFilters = imageFilter || hostFilter || severityFilter;

  const clearFilters = () => {
    setImageFilter("");
    setHostFilter("");
    setSeverityFilter("");
    setParams((prev) => ({ ...prev, page: 1 }));
  };

  return (
    <div className="px-4 py-4 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <History className="h-5 w-5 text-primary" />
          <h1 className="text-lg font-semibold">Scan History</h1>
        </div>
        {historyData && (
          <span className="text-xs text-muted-foreground">
            {historyData.total} scans
          </span>
        )}
      </div>

      {/* Filter toggle */}
      <Button
        variant="outline"
        size="sm"
        className="w-full justify-between"
        onClick={() => setShowFilters(!showFilters)}
      >
        <span className="flex items-center gap-2">
          <Search className="h-3.5 w-3.5" />
          Filters
          {hasFilters && (
            <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
              Active
            </Badge>
          )}
        </span>
        <ChevronDown className={`h-3.5 w-3.5 transition-transform ${showFilters ? "rotate-180" : ""}`} />
      </Button>

      {/* Filters */}
      {showFilters && (
        <div className="space-y-2 rounded-lg border p-3 bg-card">
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
            <Input
              placeholder="Filter by image..."
              value={imageFilter}
              onChange={(e) => {
                setImageFilter(e.target.value);
                setParams((prev) => ({ ...prev, page: 1 }));
              }}
              className="pl-8 h-9 text-sm"
            />
          </div>

          <div className="grid grid-cols-2 gap-2">
            <Select value={hostFilter || "all"} onValueChange={(v) => {
              setHostFilter(v === "all" ? "" : v);
              setParams((prev) => ({ ...prev, page: 1 }));
            }}>
              <SelectTrigger className="h-9 text-sm">
                <SelectValue placeholder="All hosts" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All hosts</SelectItem>
                {uniqueHosts.map((host) => (
                  <SelectItem key={host} value={host}>{host}</SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select value={severityFilter || "all"} onValueChange={(v) => {
              setSeverityFilter(v === "all" ? "" : v);
              setParams((prev) => ({ ...prev, page: 1 }));
            }}>
              <SelectTrigger className="h-9 text-sm">
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
          </div>

          {hasFilters && (
            <Button variant="ghost" size="sm" className="w-full" onClick={clearFilters}>
              <X className="h-3.5 w-3.5 mr-1" />
              Clear filters
            </Button>
          )}
        </div>
      )}

      {/* Results */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : !historyData?.results.length ? (
        <div className="text-center py-12 text-sm text-muted-foreground">
          No scan history found
        </div>
      ) : (
        <div className="space-y-2">
          {historyData.results.map((result) => (
            <button
              key={result.id}
              onClick={() => setSelectedScanId(result.id)}
              className="w-full text-left rounded-xl border bg-card p-3 space-y-2 active:bg-muted/50 transition-colors"
            >
              <div className="flex items-start justify-between gap-2">
                <p className="font-mono text-xs leading-tight truncate flex-1">
                  {result.image_ref}
                </p>
                <Badge variant="outline" className="text-[10px] shrink-0">
                  {result.host}
                </Badge>
              </div>

              <div className="flex items-center gap-1.5 flex-wrap">
                {result.summary.critical > 0 && (
                  <Badge className={`text-[10px] px-1.5 py-0 ${getSeverityClass("Critical")}`}>
                    C:{result.summary.critical}
                  </Badge>
                )}
                {result.summary.high > 0 && (
                  <Badge className={`text-[10px] px-1.5 py-0 ${getSeverityClass("High")}`}>
                    H:{result.summary.high}
                  </Badge>
                )}
                {result.summary.medium > 0 && (
                  <Badge className={`text-[10px] px-1.5 py-0 ${getSeverityClass("Medium")}`}>
                    M:{result.summary.medium}
                  </Badge>
                )}
                {result.summary.low > 0 && (
                  <Badge className={`text-[10px] px-1.5 py-0 ${getSeverityClass("Low")}`}>
                    L:{result.summary.low}
                  </Badge>
                )}
                {result.summary.total === 0 && (
                  <Badge variant="secondary" className="text-[10px] px-1.5 py-0">
                    Clean
                  </Badge>
                )}
              </div>

              <div className="flex items-center justify-between text-[10px] text-muted-foreground">
                <span className="capitalize">{result.scanner}</span>
                <span>{format(new Date(result.completed_at * 1000), "MMM d, HH:mm")}</span>
                <span>{(result.duration_ms / 1000).toFixed(1)}s</span>
              </div>
            </button>
          ))}
        </div>
      )}

      {/* Pagination */}
      {historyData && historyData.total_pages > 1 && (
        <div className="flex items-center justify-between pt-2">
          <span className="text-xs text-muted-foreground">
            {historyData.page} / {historyData.total_pages}
          </span>
          <div className="flex items-center gap-1">
            <Button
              variant="outline"
              size="sm"
              disabled={historyData.page <= 1}
              onClick={() => setParams((prev) => ({ ...prev, page: (prev.page ?? 1) - 1 }))}
            >
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={historyData.page >= historyData.total_pages}
              onClick={() => setParams((prev) => ({ ...prev, page: (prev.page ?? 1) + 1 }))}
            >
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}

      {/* Detail Dialog */}
      <Dialog open={!!selectedScanId} onOpenChange={(open) => !open && setSelectedScanId(null)}>
        <DialogContent className="max-w-[95vw] max-h-[85vh] overflow-y-auto p-4">
          <DialogHeader>
            <DialogTitle className="text-sm">
              {detailResult ? (
                <span className="flex items-center gap-2 flex-wrap">
                  <code className="text-xs">{detailResult.image_ref}</code>
                  <Badge variant="outline" className="text-[10px]">{detailResult.host}</Badge>
                </span>
              ) : (
                "Scan Details"
              )}
            </DialogTitle>
          </DialogHeader>

          {isDetailLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : detailResult ? (
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-2 text-xs">
                <div>
                  <span className="text-muted-foreground">Scanner:</span>{" "}
                  <span className="capitalize">{detailResult.scanner}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Duration:</span>{" "}
                  {(detailResult.duration_ms / 1000).toFixed(1)}s
                </div>
                <div className="col-span-2">
                  <span className="text-muted-foreground">Completed:</span>{" "}
                  {format(new Date(detailResult.completed_at * 1000), "MMM d, yyyy HH:mm:ss")}
                </div>
              </div>

              <ScanResultPanel result={detailResult} />
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
