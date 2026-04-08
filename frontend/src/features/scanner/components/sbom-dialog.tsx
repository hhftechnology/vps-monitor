import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { DownloadIcon, FileCheck2Icon, FileTextIcon, HistoryIcon } from "lucide-react";
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import { SBOMRegenBlockedError, downloadSBOM } from "../api/generate-sbom";
import { useGenerateSBOM, useSBOMJob } from "../hooks/use-scan-query";
import type { SBOMFormat } from "../types";

interface SBOMDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  imageRef: string;
  host: string;
}

export function SBOMDialog({ isOpen, onOpenChange, imageRef, host }: SBOMDialogProps) {
  const queryClient = useQueryClient();
  const [format, setFormat] = useState<SBOMFormat>("spdx-json");
  const [jobId, setJobId] = useState<string | null>(null);
  const [started, setStarted] = useState(false);
  const [downloading, setDownloading] = useState(false);
  const [sbomData, setSbomData] = useState<any>(null);
  const [regenBlocked, setRegenBlocked] = useState(false);

  const generateMutation = useGenerateSBOM();
  const { data: sbomJob } = useSBOMJob(jobId, started);

  const isGenerating = sbomJob && !["complete", "failed", "cancelled"].includes(sbomJob.status);
  const isComplete = sbomJob?.status === "complete";
  const isFailed = sbomJob?.status === "failed";

  useEffect(() => {
    if (isComplete && !sbomData && jobId) {
      downloadSBOM(jobId)
        .then((blob) => blob.text())
        .then((text) => JSON.parse(text))
        .then((json) => setSbomData(json))
        .catch(console.error);
    }
  }, [isComplete, sbomData, jobId]);

  const toastFiredRef = useRef(false);
  useEffect(() => {
    toastFiredRef.current = false;
  }, [jobId]);
  useEffect(() => {
    if (!isComplete || toastFiredRef.current) return;
    toastFiredRef.current = true;
    toast.success("SBOM generated and saved to history");
    queryClient.invalidateQueries({ queryKey: ["sbomedImages"] });
    queryClient.invalidateQueries({ queryKey: ["sbomHistory"] });
  }, [isComplete, queryClient]);

  const getSbomComponents = () => {
    if (!sbomData) return [];
    if (format === "cyclonedx-json" && sbomData.components) {
      return sbomData.components.map((c: any) => ({
        name: c.name,
        version: c.version,
        type: c.type,
        purl: c.purl,
      }));
    }
    if (format === "spdx-json" && sbomData.packages) {
      return sbomData.packages.map((p: any) => {
        const purlRef = p.externalRefs?.find((r: any) => r.referenceType === "purl");
        return {
          name: p.name,
          version: p.versionInfo,
          type: "package",
          purl: purlRef ? purlRef.referenceLocator : "",
        };
      });
    }
    return [];
  };

  const handleGenerate = async () => {
    try {
      const job = await generateMutation.mutateAsync({ imageRef, host, format });
      setJobId(job.id);
      setStarted(true);
    } catch (error) {
      if (error instanceof SBOMRegenBlockedError) {
        setRegenBlocked(true);
        return;
      }

      toast.error(error instanceof Error ? error.message : "Failed to generate SBOM");
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
      setSbomData(null);
      setRegenBlocked(false);
    }
    onOpenChange(open);
  };

  const components = getSbomComponents();

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className={isComplete ? "max-w-4xl max-h-[85vh] overflow-y-auto" : "max-w-md"}>
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

        {regenBlocked ? (
          <div className="space-y-4">
            <div className="flex items-center gap-3 rounded-md bg-muted p-4">
              <FileCheck2Icon className="size-5 shrink-0 text-green-500" />
              <div>
                <p className="font-medium">Already generated</p>
                <p className="text-sm text-muted-foreground">
                  This image hasn't changed since the last SBOM was generated.
                </p>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => handleClose(false)}>
                Close
              </Button>
              <Button asChild>
                <Link to="/sbom-history">
                  <HistoryIcon className="mr-2 size-4" />
                  View SBOM History
                </Link>
              </Button>
            </div>
          </div>
        ) : !started ? (
          <div className="space-y-4">
            <div className="space-y-1">
              <label htmlFor="sbom-format" className="text-sm font-medium">Format</label>
              <Select value={format} onValueChange={(value) => setFormat(value as SBOMFormat)}>
                <SelectTrigger id="sbom-format">
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
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium text-green-700 dark:text-green-400">SBOM Details</p>
                <p className="text-sm text-muted-foreground">
                  Format: {format === "spdx-json" ? "SPDX" : "CycloneDX"} JSON &bull; {components.length} components found
                </p>
              </div>
              <Button onClick={handleDownload} disabled={downloading} size="sm">
                <DownloadIcon className="mr-2 size-4" />
                {downloading ? "Downloading..." : "Export"}
              </Button>
            </div>

            <div className="border rounded-md overflow-hidden">
              <div className="max-h-[50vh] overflow-y-auto">
                <Table>
                  <TableHeader className="bg-muted/50 sticky top-0">
                    <TableRow>
                      <TableHead>Package</TableHead>
                      <TableHead>Version</TableHead>
                      <TableHead>Type</TableHead>
                      <TableHead>PURL</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {!sbomData ? (
                      <TableRow>
                        <TableCell colSpan={4} className="text-center py-8">
                          <Spinner className="size-5 mx-auto" />
                        </TableCell>
                      </TableRow>
                    ) : components.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={4} className="text-center py-8 text-muted-foreground">
                          No components found.
                        </TableCell>
                      </TableRow>
                    ) : (
                      components.map((c: any, i: number) => (
                        <TableRow key={i}>
                          <TableCell className="font-medium">{c.name}</TableCell>
                          <TableCell>{c.version}</TableCell>
                          <TableCell>
                            <Badge variant="outline" className="capitalize">
                              {c.type || "unknown"}
                            </Badge>
                          </TableCell>
                          <TableCell className="font-mono text-xs text-muted-foreground break-all">
                            {c.purl}
                          </TableCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <Button variant="outline" onClick={() => handleClose(false)}>
                Close
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
