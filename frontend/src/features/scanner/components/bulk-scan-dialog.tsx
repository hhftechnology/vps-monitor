import { useMemo, useState } from "react";
import { ShieldCheckIcon, SkipForwardIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { useCancelScan, useScanJob, useStartBulkScan } from "../hooks/use-scan-query";
import { ScanResultsExport } from "./scan-results-export";
import { ScanResultsSummary } from "./scan-results-summary";
import { ScanResultsTable } from "./scan-results-table";
import type { ScanResult, ScannerType, SeveritySummary } from "../types";

interface BulkScanDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function BulkScanDialog({ isOpen, onOpenChange }: BulkScanDialogProps) {
  const [scanner, setScanner] = useState<ScannerType>("grype");
  const [jobId, setJobId] = useState<string | null>(null);
  const [started, setStarted] = useState(false);
  const [selectedResult, setSelectedResult] = useState<ScanResult | null>(null);

  const startBulkScanMutation = useStartBulkScan();
  const cancelScanMutation = useCancelScan();
  const { data: jobData } = useScanJob(jobId, started);

  const bulkJob = jobData?.bulkJob;
  const isScanning = bulkJob && !["complete", "failed", "cancelled"].includes(bulkJob.status);
  const isComplete = bulkJob?.status === "complete";

  const progress = bulkJob
    ? ((bulkJob.completed + bulkJob.failed) / Math.max(bulkJob.total_images, 1)) * 100
    : 0;

  const handleStart = async () => {
    try {
      const newJob = await startBulkScanMutation.mutateAsync({ scanner });
      setJobId(newJob.id);
      setStarted(true);
      setSelectedResult(null);
    } catch {
      // mutation handles errors
    }
  };

  const handleCancel = () => {
    if (jobId) {
      cancelScanMutation.mutate(jobId);
    }
  };

  const handleClose = (open: boolean) => {
    if (!open) {
      setJobId(null);
      setStarted(false);
      setSelectedResult(null);
    }
    onOpenChange(open);
  };

  const aggregateSummary = useMemo(() => {
    const summary: SeveritySummary = {
      critical: 0,
      high: 0,
      medium: 0,
      low: 0,
      negligible: 0,
      unknown: 0,
      total: 0,
    };

    if (bulkJob?.jobs) {
      for (const job of bulkJob.jobs) {
        if (job.result) {
          summary.critical += job.result.summary.critical;
          summary.high += job.result.summary.high;
          summary.medium += job.result.summary.medium;
          summary.low += job.result.summary.low;
          summary.negligible += job.result.summary.negligible;
          summary.unknown += job.result.summary.unknown;
          summary.total += job.result.summary.total;
        }
      }
    }
    
    return summary;
  }, [bulkJob?.jobs]);

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ShieldCheckIcon className="size-5" />
            Bulk vulnerability scan
          </DialogTitle>
          <DialogDescription>
            Scan all Docker images for known vulnerabilities.
          </DialogDescription>
        </DialogHeader>

        {!started ? (
          <div className="space-y-4">
            <div className="space-y-1">
              <label htmlFor="scanner-select" className="text-sm font-medium">Scanner</label>
              <Select value={scanner} onValueChange={(value) => setScanner(value as ScannerType)}>
                <SelectTrigger id="scanner-select" className="w-40">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="grype">Grype</SelectItem>
                  <SelectItem value="trivy">Trivy</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => handleClose(false)}>
                Cancel
              </Button>
              <Button onClick={handleStart} disabled={startBulkScanMutation.isPending}>
                {startBulkScanMutation.isPending ? (
                  <>
                    <Spinner className="mr-2 size-4" />
                    Starting...
                  </>
                ) : (
                  "Scan All Images"
                )}
              </Button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {isScanning && bulkJob && (
              <>
                <div className="flex items-center gap-3">
                  <Spinner className="size-5" />
                  <p className="font-medium">
                    Scanning images... ({bulkJob.completed + bulkJob.failed}/{bulkJob.total_images})
                  </p>
                </div>
                <Progress value={progress} className="h-2" />
              </>
            )}

            {isComplete && bulkJob && (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <p className="font-medium">
                    Scan complete - {bulkJob.total_images} images
                  </p>
                  <div className="flex gap-2">
                    <Badge variant="outline">{bulkJob.completed} succeeded</Badge>
                    {bulkJob.failed > 0 && (
                      <Badge variant="destructive">{bulkJob.failed} failed</Badge>
                    )}
                  </div>
                </div>
                {bulkJob.jobs.filter((j) => j.error?.includes("image_unchanged")).length > 0 && (
                  <div className="flex items-center gap-2 rounded-md bg-muted p-2 text-sm text-muted-foreground">
                    <SkipForwardIcon className="size-4 shrink-0" />
                    {bulkJob.jobs.filter((j) => j.error?.includes("image_unchanged")).length} images skipped (unchanged since last scan)
                  </div>
                )}
                <ScanResultsSummary summary={aggregateSummary} />
              </div>
            )}

            {bulkJob && bulkJob.jobs.length > 0 && (
              <ScrollArea className="h-[300px] rounded-md border">
                <div className="p-3 space-y-2">
                  {bulkJob.jobs.map((job) => (
                    <div
                      key={job.id}
                      className="flex items-center justify-between gap-3 rounded bg-muted/50 p-2"
                    >
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-mono">{job.image_ref}</p>
                        <p className="text-xs text-muted-foreground">{job.host}</p>
                      </div>
                      <div className="ml-3 flex items-center gap-2">
                        {job.status === "complete" && job.result ? (
                          <>
                            <Badge
                              variant={job.result.summary.total > 0 ? "destructive" : "outline"}
                              className={job.result.summary.total === 0 ? "border-green-500 text-green-500" : ""}
                            >
                              {job.result.summary.total} vulns
                            </Badge>
                            <Button variant="ghost" size="sm" onClick={() => setSelectedResult(job.result ?? null)}>
                              View
                            </Button>
                          </>
                        ) : job.status === "failed" && job.error?.includes("image_unchanged") ? (
                          <Badge variant="secondary">Skipped</Badge>
                        ) : job.status === "failed" ? (
                          <Badge variant="destructive">Failed</Badge>
                        ) : job.status === "cancelled" ? (
                          <Badge variant="secondary">Cancelled</Badge>
                        ) : (
                          <Spinner className="size-4" />
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </ScrollArea>
            )}

            {selectedResult && (
              <div className="space-y-4 rounded-md border p-4">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <p className="font-medium">{selectedResult.image_ref}</p>
                    <p className="text-sm text-muted-foreground">{selectedResult.host}</p>
                  </div>
                  <ScanResultsExport result={selectedResult} />
                </div>
                <ScanResultsSummary summary={selectedResult.summary} />
                <Tabs defaultValue="results">
                  <TabsList>
                    <TabsTrigger value="results">Results</TabsTrigger>
                  </TabsList>
                  <TabsContent value="results">
                    <ScanResultsTable vulnerabilities={selectedResult.vulnerabilities} />
                  </TabsContent>
                </Tabs>
              </div>
            )}

            <div className="flex justify-end gap-2">
              {isScanning && (
                <Button variant="outline" onClick={handleCancel}>
                  Cancel
                </Button>
              )}
              {(isComplete || bulkJob?.status === "failed" || bulkJob?.status === "cancelled") && (
                <Button variant="outline" onClick={() => handleClose(false)}>
                  Close
                </Button>
              )}
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
