import { useCallback, useRef, useState } from "react";
import { CheckIcon, DownloadIcon } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Spinner } from "@/components/ui/spinner";

import { pullImage } from "../api/pull-image";
import type { ImagePullProgress } from "../types";

// Need to create the dialog component - let me check if it exists
interface ImagePullDialogProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  hosts: string[];
  selectedHosts: string[];
  onSelectedHostsChange: (hosts: string[]) => void;
}

export function ImagePullDialog({
  isOpen,
  onOpenChange,
  hosts,
  selectedHosts,
  onSelectedHostsChange,
}: ImagePullDialogProps) {
  const queryClient = useQueryClient();
  const [imageName, setImageName] = useState("");
  const [isPulling, setIsPulling] = useState(false);
  const [progress, setProgress] = useState<ImagePullProgress[]>([]);
  const abortControllerRef = useRef<AbortController | null>(null);

  const toggleHost = (host: string) => {
    if (selectedHosts.includes(host)) {
      onSelectedHostsChange(selectedHosts.filter((h) => h !== host));
    } else {
      onSelectedHostsChange([...selectedHosts, host]);
    }
  };

  const handlePull = useCallback(async () => {
    if (!imageName.trim() || selectedHosts.length === 0) return;

    setIsPulling(true);
    setProgress([]);

    const abortController = new AbortController();
    abortControllerRef.current = abortController;

    try {
      // Pull to each selected host sequentially
      for (const host of selectedHosts) {
        setProgress((prev) => [
          ...prev,
          { status: `Starting pull on ${host}...` },
        ]);

        try {
          for await (const item of pullImage(
            { imageName: imageName.trim(), host },
            abortController.signal
          )) {
            setProgress((prev) => [...prev, item]);
          }
          setProgress((prev) => [
            ...prev,
            { status: `Completed on ${host}` },
          ]);
        } catch (err) {
          if ((err as Error).name === "AbortError") {
            setProgress((prev) => [...prev, { status: "Pull cancelled" }]);
            break;
          }
          setProgress((prev) => [
            ...prev,
            {
              status: `Error on ${host}: ${err instanceof Error ? err.message : "Unknown error"}`,
            },
          ]);
        }
      }

      toast.success("Image pull completed");
      queryClient.invalidateQueries({ queryKey: ["images"] });
    } catch (err) {
      if ((err as Error).name !== "AbortError") {
        toast.error(
          `Failed to pull image: ${err instanceof Error ? err.message : "Unknown error"}`
        );
      }
    } finally {
      setIsPulling(false);
      abortControllerRef.current = null;
    }
  }, [imageName, selectedHosts, queryClient]);

  const handleCancel = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
  };

  const handleClose = (open: boolean) => {
    if (!open && isPulling) {
      handleCancel();
    }
    if (!open) {
      setImageName("");
      setProgress([]);
    }
    onOpenChange(open);
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Pull Docker Image</DialogTitle>
          <DialogDescription>
            Pull an image from a registry to one or more hosts
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="image-name">Image Name</Label>
            <Input
              id="image-name"
              placeholder="e.g., nginx:latest, ubuntu:22.04"
              value={imageName}
              onChange={(e) => setImageName(e.target.value)}
              disabled={isPulling}
            />
          </div>

          <div className="space-y-2">
            <Label>Target Hosts</Label>
            <div className="flex flex-wrap gap-2">
              {hosts.map((host) => (
                <Badge
                  key={host}
                  variant={selectedHosts.includes(host) ? "default" : "outline"}
                  className="cursor-pointer"
                  onClick={() => !isPulling && toggleHost(host)}
                >
                  {selectedHosts.includes(host) && (
                    <CheckIcon className="mr-1 size-3" />
                  )}
                  {host}
                </Badge>
              ))}
            </div>
          </div>

          {progress.length > 0 && (
            <div className="space-y-2">
              <Label>Progress</Label>
              <ScrollArea className="h-[200px] w-full rounded-md border bg-muted/30 p-3">
                <div className="space-y-1 font-mono text-xs">
                  {progress.map((item, i) => (
                    <div key={i} className="flex gap-2">
                      {item.id && (
                        <span className="text-muted-foreground shrink-0">
                          [{item.id}]
                        </span>
                      )}
                      <span>{item.status}</span>
                      {item.progress && (
                        <span className="text-muted-foreground">
                          {item.progress}
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              </ScrollArea>
            </div>
          )}
        </div>

        <DialogFooter>
          {isPulling ? (
            <Button variant="destructive" onClick={handleCancel}>
              Cancel
            </Button>
          ) : (
            <>
              <Button variant="outline" onClick={() => handleClose(false)}>
                Close
              </Button>
              <Button
                onClick={handlePull}
                disabled={!imageName.trim() || selectedHosts.length === 0}
              >
                <DownloadIcon className="mr-2 size-4" />
                Pull Image
              </Button>
            </>
          )}
        </DialogFooter>

        {isPulling && (
          <div className="absolute inset-0 flex items-center justify-center bg-background/50">
            <div className="flex items-center gap-2">
              <Spinner className="size-4" />
              <span>Pulling image...</span>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
