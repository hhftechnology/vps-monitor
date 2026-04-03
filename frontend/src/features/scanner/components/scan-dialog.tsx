import { useState } from "react";
import { ShieldAlertIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { useStartScan, useScanJob, useCancelScan } from "../hooks/use-scan-query";
import { ScanResultsSummary } from "./scan-results-summary";
import { ScanResultsTable } from "./scan-results-table";
import { ScanResultsExport } from "./scan-results-export";
import type { ScannerType } from "../types";

interface ScanDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  imageRef: string;
  host: string;
}

export function ScanDialog({ isOpen, onOpenChange, imageRef, host }: ScanDialogProps) {
  const [scanner, setScanner] = useState<ScannerType>("grype");
  const [jobId, setJobId] = useState<string | null>(null);
  const [started, setStarted] = useState(false);

  const startScanMutation = useStartScan();
  const cancelScanMutation = useCancelScan();
  const { data: jobData } = useScanJob(jobId, started);

  const job = jobData?.job;
  const isScanning = job && !["complete", "failed", "cancelled"].includes(job.status);
  const isComplete = job?.status === "complete";
  const isFailed = job?.status === "failed" || job?.status === "cancelled";

  const handleStartScan = async () => {
    try {
      const newJob = await startScanMutation.mutateAsync({ imageRef, host, scanner });
      setJobId(newJob.id);
      setStarted(true);
    } catch {
      // error handled by mutation
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
    }
    onOpenChange(open);
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="max-w-4xl max-h-[85vh] overflow-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ShieldAlertIcon className="size-5" />
            Vulnerability scan
            <Badge variant="outline" className="font-mono">
              {imageRef}
            </Badge>
          </DialogTitle>
          <DialogDescription>
            Scan this image for known vulnerabilities using {scanner === "grype" ? "Grype" : "Trivy"}.
          </DialogDescription>
        </DialogHeader>

        {!started ? (
          <div className="space-y-4">
            <div className="flex items-center gap-4">
              <div className="space-y-1">
                <label htmlFor="scanner-select" className="text-sm font-medium">Scanner</label>
                <Select value={scanner} onValueChange={(v) => setScanner(v as ScannerType)}>
                  <SelectTrigger id="scanner-select" className="w-40">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="grype">Grype</SelectItem>
                    <SelectItem value="trivy">Trivy</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => handleClose(false)}>
                Cancel
              </Button>
              <Button onClick={handleStartScan} disabled={startScanMutation.isPending}>
                {startScanMutation.isPending ? (
                  <>
                    <Spinner className="mr-2 size-4" />
                    Starting...
                  </>
                ) : (
                  "Start Scan"
                )}
              </Button>
            </div>
          </div>
        ) : isScanning ? (
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <Spinner className="size-5" />
              <div>
                <p className="font-medium">Scanning for vulnerabilities...</p>
                <p className="text-sm text-muted-foreground">
                  {job?.progress || `Status: ${job?.status}`}
                </p>
              </div>
            </div>

            <div className="rounded-md bg-muted p-3">
              <p className="text-sm font-mono text-muted-foreground">
                {job?.progress || "Initializing scanner..."}
              </p>
            </div>

            <div className="flex justify-end">
              <Button variant="outline" onClick={handleCancel}>
                Cancel Scan
              </Button>
            </div>
          </div>
        ) : isFailed ? (
          <div className="space-y-4">
            <div className="rounded-md bg-destructive/10 p-4">
              <p className="font-medium text-destructive">Scan failed</p>
              <p className="text-sm text-muted-foreground mt-1">
                {job?.error || "An unknown error occurred"}
              </p>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => handleClose(false)}>
                Close
              </Button>
              <Button onClick={() => { setStarted(false); setJobId(null); }}>
                Retry
              </Button>
            </div>
          </div>
        ) : isComplete && job?.result ? (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  {job.result.summary.total > 0 ? (
                    <Badge variant="destructive">
                      {job.result.summary.total} vulnerabilities
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="border-green-500 text-green-500">
                      No vulnerabilities
                    </Badge>
                  )}
                  <span className="text-sm text-muted-foreground">
                    {(job.result.duration_ms / 1000).toFixed(1)}s
                  </span>
                </div>
                <ScanResultsSummary summary={job.result.summary} />
              </div>
              <ScanResultsExport result={job.result} />
            </div>

            <Tabs defaultValue="results">
              <TabsList>
                <TabsTrigger value="results">
                  Scan results
                  {job.result.summary.total > 0 && (
                    <Badge variant="destructive" className="ml-2">
                      {job.result.summary.total}
                    </Badge>
                  )}
                </TabsTrigger>
              </TabsList>
              <TabsContent value="results">
                <ScanResultsTable vulnerabilities={job.result.vulnerabilities} />
              </TabsContent>
            </Tabs>

            <div className="flex justify-end">
              <Button variant="outline" onClick={() => handleClose(false)}>
                Close
              </Button>
            </div>
          </div>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
