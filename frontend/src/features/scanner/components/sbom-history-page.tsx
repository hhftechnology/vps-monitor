import type { Dispatch, SetStateAction } from "react";
import { useState } from "react";
import { format } from "date-fns";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  DownloadIcon,
  FileTextIcon,
  SearchIcon,
  Trash2Icon,
  XIcon,
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

import { downloadSBOMHistoryFile } from "../api/get-sbom-history";
import {
  useDeleteSBOMHistory,
  useSBOMHistory,
  useSBOMHistoryDetail,
  useSBOMedImages,
} from "../hooks/use-scan-query";
import type { SBOMComponent, SBOMFormat, SBOMHistoryQueryParams } from "../types";

type FormatFilter = SBOMFormat | "";

function downloadBlob(blob: Blob, filename: string) {
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  window.URL.revokeObjectURL(url);
}

function toggleSort(
  setParams: Dispatch<SetStateAction<SBOMHistoryQueryParams>>,
  sortBy: NonNullable<SBOMHistoryQueryParams["sort_by"]>
) {
  setParams((prev) => ({
    ...prev,
    sort_by: sortBy,
    sort_dir: prev.sort_dir === "desc" ? "asc" : "desc",
  }));
}

function getAriaSort(
  params: SBOMHistoryQueryParams,
  sortBy: NonNullable<SBOMHistoryQueryParams["sort_by"]>
): "none" | "ascending" | "descending" {
  if (params.sort_by !== sortBy) return "none";
  return params.sort_dir === "asc" ? "ascending" : "descending";
}

function ComponentsTable({ components }: { components: SBOMComponent[] }) {
  return (
    <div className="border rounded-md overflow-hidden">
      <div className="max-h-[50vh] overflow-y-auto">
        <Table>
          <TableHeader className="bg-muted/50 sticky top-0">
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Version</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>PURL</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {components.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center py-8 text-muted-foreground">
                  No components found.
                </TableCell>
              </TableRow>
            ) : (
              components.map((component, index) => (
                <TableRow key={`${component.name}-${component.version}-${index}`}>
                  <TableCell className="font-medium">{component.name}</TableCell>
                  <TableCell>{component.version || "-"}</TableCell>
                  <TableCell>
                    <Badge variant="outline" className="capitalize">
                      {component.type || "unknown"}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground break-all">
                    {component.purl || "-"}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

export function SBOMHistoryPage() {
  const [params, setParams] = useState<SBOMHistoryQueryParams>({
    page: 1,
    page_size: 20,
    sort_by: "completed_at",
    sort_dir: "desc",
  });
  const [imageFilter, setImageFilter] = useState("");
  const [hostFilter, setHostFilter] = useState("");
  const [formatFilter, setFormatFilter] = useState<FormatFilter>("");
  const [selectedSBOMId, setSelectedSBOMId] = useState<string | null>(null);

  const { data: historyData, isLoading } = useSBOMHistory({
    ...params,
    image: imageFilter || undefined,
    host: hostFilter || undefined,
    format: formatFilter || undefined,
  });
  const { data: sbomedImages } = useSBOMedImages();
  const { data: detailResult, isLoading: isDetailLoading } = useSBOMHistoryDetail(selectedSBOMId);
  const deleteMutation = useDeleteSBOMHistory();

  const uniqueHosts = Array.from(new Set(sbomedImages?.map((img) => img.host) ?? []));
  const hasFilters = imageFilter || hostFilter || formatFilter;

  const clearFilters = () => {
    setImageFilter("");
    setHostFilter("");
    setFormatFilter("");
    setParams((prev) => ({ ...prev, page: 1 }));
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <FileTextIcon className="size-6" />
          <h1 className="text-2xl font-bold">SBOM History</h1>
        </div>
        {historyData && (
          <p className="text-sm text-muted-foreground">
            {historyData.total} total SBOMs
          </p>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-[300px]">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            placeholder="Filter by image..."
            value={imageFilter}
            onChange={(event) => {
              setImageFilter(event.target.value);
              setParams((prev) => ({ ...prev, page: 1 }));
            }}
            className="pl-9"
          />
        </div>

        <Select
          value={hostFilter || "all"}
          onValueChange={(value) => {
            setHostFilter(value === "all" ? "" : value);
            setParams((prev) => ({ ...prev, page: 1 }));
          }}
        >
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

        <Select
          value={formatFilter || "all"}
          onValueChange={(value) => {
            setFormatFilter(value === "all" ? "" : (value as SBOMFormat));
            setParams((prev) => ({ ...prev, page: 1 }));
          }}
        >
          <SelectTrigger className="w-[180px]">
            <SelectValue placeholder="All formats" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All formats</SelectItem>
            <SelectItem value="spdx-json">SPDX JSON</SelectItem>
            <SelectItem value="cyclonedx-json">CycloneDX JSON</SelectItem>
          </SelectContent>
        </Select>

        {hasFilters && (
          <Button variant="ghost" size="sm" onClick={clearFilters}>
            <XIcon className="size-4 mr-1" />
            Clear
          </Button>
        )}
      </div>

      <div className="border rounded-lg">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Image</TableHead>
              <TableHead>Host</TableHead>
              <TableHead>Format</TableHead>
              <TableHead aria-sort={getAriaSort(params, "component_count")}>
                <button
                  type="button"
                  className="inline-flex items-center gap-1 font-medium"
                  onClick={() => toggleSort(setParams, "component_count")}
                >
                  Components
                  {params.sort_by === "component_count" && (params.sort_dir === "desc" ? "\u2193" : "\u2191")}
                </button>
              </TableHead>
              <TableHead aria-sort={getAriaSort(params, "completed_at")}>
                <button
                  type="button"
                  className="inline-flex items-center gap-1 font-medium"
                  onClick={() => toggleSort(setParams, "completed_at")}
                >
                  Date
                  {params.sort_by === "completed_at" && (params.sort_dir === "desc" ? "\u2193" : "\u2191")}
                </button>
              </TableHead>
              <TableHead>Duration</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                  Loading SBOM history...
                </TableCell>
              </TableRow>
            ) : !historyData?.results.length ? (
              <TableRow>
                <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                  No SBOM history found
                </TableCell>
              </TableRow>
            ) : (
              historyData.results.map((result) => (
                <TableRow key={result.id}>
                  <TableCell className="font-mono text-sm max-w-[250px] truncate">
                    <button
                      type="button"
                      className="max-w-full truncate text-left underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                      onClick={() => setSelectedSBOMId(result.id)}
                    >
                      {result.image_ref}
                    </button>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{result.host}</Badge>
                  </TableCell>
                  <TableCell className="capitalize">{result.format}</TableCell>
                  <TableCell>{result.component_count}</TableCell>
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
                        title="Download SBOM JSON"
                        onClick={() => {
                          downloadSBOMHistoryFile(result.id)
                            .then(({ blob, filename }) => downloadBlob(blob, filename))
                            .catch((error) => {
                              console.error("Failed to download SBOM:", error);
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
                        onClick={(event) => {
                          event.stopPropagation();
                          if (confirm("Are you sure you want to delete this SBOM result?")) {
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

      <Dialog open={!!selectedSBOMId} onOpenChange={(open) => !open && setSelectedSBOMId(null)}>
        <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {detailResult ? (
                <span className="flex items-center gap-2">
                  SBOM: <code className="text-sm">{detailResult.image_ref}</code>
                  <Badge variant="outline">{detailResult.host}</Badge>
                </span>
              ) : (
                "SBOM Details"
              )}
            </DialogTitle>
          </DialogHeader>

          {isDetailLoading ? (
            <div className="py-8 text-center text-muted-foreground">Loading SBOM details...</div>
          ) : detailResult ? (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-muted-foreground">Format:</span>{" "}
                  <span className="capitalize">{detailResult.format}</span>
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
                  <span className="text-muted-foreground">Components:</span>{" "}
                  {detailResult.component_count}
                </div>
              </div>

              <ComponentsTable components={detailResult.components ?? []} />
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}
