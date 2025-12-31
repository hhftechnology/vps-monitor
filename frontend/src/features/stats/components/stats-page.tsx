import { useMemo, useState } from "react";
import { PlayIcon, RefreshCcwIcon, SquareIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { useContainersQuery } from "@/features/containers/hooks/use-containers-query";

import { ContainerStatsCard } from "./container-stats-card";

export function StatsPage() {
  const { data, isLoading, error, refetch, isRefetching } =
    useContainersQuery();
  const [searchText, setSearchText] = useState("");
  const [enabledContainers, setEnabledContainers] = useState<Set<string>>(
    new Set()
  );

  const containers = data?.containers ?? [];

  // Filter to only running containers (stats only available for running containers)
  const runningContainers = useMemo(() => {
    return containers.filter(
      (c) => c.state.toLowerCase() === "running"
    );
  }, [containers]);

  // Filter by search text
  const filteredContainers = useMemo(() => {
    if (!searchText) return runningContainers;
    const search = searchText.toLowerCase();
    return runningContainers.filter(
      (c) =>
        c.names.some((n) => n.toLowerCase().includes(search)) ||
        c.image.toLowerCase().includes(search) ||
        c.id.toLowerCase().startsWith(search)
    );
  }, [runningContainers, searchText]);

  const handleToggleContainer = (containerId: string) => {
    setEnabledContainers((prev) => {
      const next = new Set(prev);
      if (next.has(containerId)) {
        next.delete(containerId);
      } else {
        next.add(containerId);
      }
      return next;
    });
  };

  const handleEnableAll = () => {
    setEnabledContainers(new Set(filteredContainers.map((c) => c.id)));
  };

  const handleDisableAll = () => {
    setEnabledContainers(new Set());
  };

  const enabledCount = filteredContainers.filter((c) =>
    enabledContainers.has(c.id)
  ).length;
  const allEnabled = enabledCount === filteredContainers.length && filteredContainers.length > 0;

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Spinner className="mr-2 size-4" />
          Loading containers...
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-destructive">
          Failed to load containers: {error.message}
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <CardTitle>Container Stats</CardTitle>
              <p className="text-sm text-muted-foreground">
                Real-time resource monitoring for running containers
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="outline">
                {enabledCount} / {filteredContainers.length} streaming
              </Badge>
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
            </div>
          </div>
          <div className="flex items-center gap-4 pt-4">
            <Input
              placeholder="Search containers..."
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              className="max-w-sm"
            />
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={allEnabled ? handleDisableAll : handleEnableAll}
              >
                {allEnabled ? (
                  <>
                    <SquareIcon className="mr-2 size-4" />
                    Stop All
                  </>
                ) : (
                  <>
                    <PlayIcon className="mr-2 size-4" />
                    Start All
                  </>
                )}
              </Button>
            </div>
          </div>
        </CardHeader>
      </Card>

      {runningContainers.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            No running containers found. Stats are only available for running
            containers.
          </CardContent>
        </Card>
      ) : filteredContainers.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            No containers match your search
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {filteredContainers.map((container) => (
            <ContainerStatsCard
              key={container.id}
              container={container}
              isEnabled={enabledContainers.has(container.id)}
              onToggle={() => handleToggleContainer(container.id)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
