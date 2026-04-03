import { useState } from "react";
import { DownloadIcon, FileTextIcon } from "lucide-react";
import { toast } from "sonner";

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

import { downloadSBOM } from "../api/generate-sbom";
import { useGenerateSBOM, useSBOMJob } from "../hooks/use-scan-query";
import type { SBOMFormat } from "../types";

interface SBOMDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  imageRef: string;
  host: string;
}

export function SBOMDialog({ isOpen, onOpenChange, imageRef, host }: SBOMDialogProps) {
  const [format, setFormat] = useState<SBOMFormat>("spdx-json");
  const [jobId, setJobId] = useState<string | null>(null);
  const [started, setStarted] = useState(false);
  const [downloading, setDownloading] = useState(false);

  const generateMutation = useGenerateSBOM();
  const { data: sbomJob } = useSBOMJob(jobId, started);

  const isGenerating = sbomJob && !["complete", "failed", "cancelled"].includes(sbomJob.status);
  const isComplete = sbomJob?.status === "complete";
  const isFailed = sbomJob?.status === "failed";

  const handleGenerate = async () => {
    try {
      const job = await generateMutation.mutateAsync({ imageRef, host, format });
      setJobId(job.id);
      setStarted(true);
    } catch {
      // mutation handles errors
    }
  };

  const handleDownload = async () => {
    if (!jobId) return;

    try {
      setDownloading(true);
      const blob = await downloadSBOM(jobId);
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `sbom-${imageRef.replace(/[/:]/g, "_")}.json`;
      link.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to download SBOM");
    } finally {
      setDownloading(false);
    }
  };

  const handleClose = (open: boolean) => {
    if (!open) {
      setJobId(null);
      setStarted(false);
      setDownloading(false);
    }
    onOpenChange(open);
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FileTextIcon className="size-5" />
            Generate SBOM
          </DialogTitle>
          <DialogDescription>
            Generate a Software Bill of Materials for{" "}
            <Badge variant="outline" className="font-mono">
              {imageRef}
            </Badge>
          </DialogDescription>
        </DialogHeader>

        {!started ? (
          <div className="space-y-4">
            <div className="space-y-1">
              <label className="text-sm font-medium">Format</label>
              <Select value={format} onValueChange={(value) => setFormat(value as SBOMFormat)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="spdx-json">SPDX (JSON)</SelectItem>
                  <SelectItem value="cyclonedx-json">CycloneDX (JSON)</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => handleClose(false)}>
                Cancel
              </Button>
              <Button onClick={handleGenerate} disabled={generateMutation.isPending}>
                {generateMutation.isPending ? (
                  <>
                    <Spinner className="mr-2 size-4" />
                    Starting...
                  </>
                ) : (
                  "Generate"
                )}
              </Button>
            </div>
          </div>
        ) : isGenerating ? (
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <Spinner className="size-5" />
              <div>
                <p className="font-medium">Generating SBOM...</p>
                <p className="text-sm text-muted-foreground">
                  This may take a minute for large images.
                </p>
              </div>
            </div>
          </div>
        ) : isComplete ? (
          <div className="space-y-4">
            <div className="rounded-md bg-green-50 dark:bg-green-950 p-4">
              <p className="font-medium text-green-700 dark:text-green-400">SBOM generated successfully</p>
              <p className="text-sm text-muted-foreground mt-1">
                Format: {format === "spdx-json" ? "SPDX" : "CycloneDX"} JSON
              </p>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => handleClose(false)}>
                Close
              </Button>
              <Button onClick={handleDownload} disabled={downloading}>
                <DownloadIcon className="mr-2 size-4" />
                {downloading ? "Downloading..." : "Download SBOM"}
              </Button>
            </div>
          </div>
        ) : isFailed ? (
          <div className="space-y-4">
            <div className="rounded-md bg-destructive/10 p-4">
              <p className="font-medium text-destructive">SBOM generation failed</p>
              <p className="text-sm text-muted-foreground mt-1">
                {sbomJob?.error || "An unknown error occurred"}
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
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
