import { useMemo, useState } from "react";
import {
  DownloadIcon,
  RefreshCcwIcon,
  SearchIcon,
  Trash2Icon,
} from "lucide-react";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import { useImagesQuery, useRemoveImageMutation } from "../hooks/use-images-query";
import { ImagePullDialog } from "./image-pull-dialog";
import type { ImageInfo } from "../types";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / k ** i).toFixed(1)} ${sizes[i]}`;
}

function formatDate(timestamp: number): string {
  const date = new Date(timestamp * 1000);
  return date.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function getImageDisplayName(image: ImageInfo): string {
  if (image.repo_tags && image.repo_tags.length > 0) {
    return image.repo_tags[0];
  }
  return image.id.replace("sha256:", "").slice(0, 12);
}

export function ImagesTable() {
  const { data, isLoading, error, refetch, isRefetching } = useImagesQuery();
  const removeImageMutation = useRemoveImageMutation();

  const [searchText, setSearchText] = useState("");
  const [isPullDialogOpen, setIsPullDialogOpen] = useState(false);
  const [selectedHosts, setSelectedHosts] = useState<string[]>([]);
  const [imageToDelete, setImageToDelete] = useState<{
    image: ImageInfo;
    host: string;
  } | null>(null);

  // Images already come as flat array with host field
  const allImages = useMemo(() => {
    if (!data?.images) return [];
    return data.images;
  }, [data?.images]);

  // Get unique hosts for pull dialog
  const hosts = useMemo(() => {
    if (!data?.images) return [];
    const uniqueHosts = new Set(data.images.map((img) => img.host));
    return Array.from(uniqueHosts);
  }, [data?.images]);

  // Filter images by search
  const filteredImages = useMemo(() => {
    if (!searchText) return allImages;
    const search = searchText.toLowerCase();
    return allImages.filter((img) => {
      const name = getImageDisplayName(img).toLowerCase();
      const id = img.id.toLowerCase();
      return name.includes(search) || id.includes(search);
    });
  }, [allImages, searchText]);

  const handleDelete = async () => {
    if (!imageToDelete) return;

    try {
      await removeImageMutation.mutateAsync({
        imageId: imageToDelete.image.id,
        host: imageToDelete.host,
        force: false,
      });
      toast.success("Image removed successfully");
    } catch (err) {
      toast.error(
        `Failed to remove image: ${err instanceof Error ? err.message : "Unknown error"}`
      );
    } finally {
      setImageToDelete(null);
    }
  };

  const openPullDialog = () => {
    setSelectedHosts(hosts.length > 0 ? [hosts[0]] : []);
    setIsPullDialogOpen(true);
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Spinner className="mr-2 size-4" />
          Loading images...
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-destructive">
          Failed to load images: {error.message}
        </CardContent>
      </Card>
    );
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Docker Images</CardTitle>
            <div className="flex items-center gap-2">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => refetch()}
                    disabled={isRefetching}
                  >
                    <RefreshCcwIcon
                      className={`size-4 ${isRefetching ? "animate-spin" : ""}`}
                    />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Refresh</TooltipContent>
              </Tooltip>
              {!data?.readOnly && (
                <Button variant="default" size="sm" onClick={openPullDialog}>
                  <DownloadIcon className="mr-2 size-4" />
                  Pull Image
                </Button>
              )}
            </div>
          </div>
          <div className="relative mt-4">
            <SearchIcon className="absolute left-2 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
            <Input
              placeholder="Search images..."
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              className="pl-8"
            />
          </div>
        </CardHeader>
        <CardContent>
          {filteredImages.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              {searchText ? "No images match your search" : "No images found"}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Image</TableHead>
                  <TableHead>Host</TableHead>
                  <TableHead>Size</TableHead>
                  <TableHead>Created</TableHead>
                  {!data?.readOnly && (
                    <TableHead className="w-[80px]">Actions</TableHead>
                  )}
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredImages.map((image) => (
                  <TableRow key={`${image.host}-${image.id}`}>
                    <TableCell>
                      <div className="flex flex-col gap-1">
                        <span className="font-medium">
                          {getImageDisplayName(image)}
                        </span>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="text-xs text-muted-foreground font-mono cursor-help">
                              {image.id.replace("sha256:", "").slice(0, 12)}
                            </span>
                          </TooltipTrigger>
                          <TooltipContent className="max-w-md">
                            {image.id}
                          </TooltipContent>
                        </Tooltip>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{image.host}</Badge>
                    </TableCell>
                    <TableCell>{formatBytes(image.size)}</TableCell>
                    <TableCell>{formatDate(image.created)}</TableCell>
                    {!data?.readOnly && (
                      <TableCell>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              onClick={() =>
                                setImageToDelete({
                                  image,
                                  host: image.host,
                                })
                              }
                            >
                              <Trash2Icon className="size-4 text-destructive" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Remove image</TooltipContent>
                        </Tooltip>
                      </TableCell>
                    )}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <ImagePullDialog
        isOpen={isPullDialogOpen}
        onOpenChange={setIsPullDialogOpen}
        hosts={hosts}
        selectedHosts={selectedHosts}
        onSelectedHostsChange={setSelectedHosts}
      />

      <AlertDialog
        open={!!imageToDelete}
        onOpenChange={(open) => !open && setImageToDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove Image</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to remove{" "}
              <span className="font-medium">
                {imageToDelete && getImageDisplayName(imageToDelete.image)}
              </span>{" "}
              from <span className="font-medium">{imageToDelete?.host}</span>?
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {removeImageMutation.isPending ? (
                <>
                  <Spinner className="mr-2 size-4" />
                  Removing...
                </>
              ) : (
                "Remove"
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
