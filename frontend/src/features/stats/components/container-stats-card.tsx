import {
  ActivityIcon,
  CpuIcon,
  HardDriveIcon,
  MemoryStickIcon,
  NetworkIcon,
  PlayIcon,
  SquareIcon,
} from "lucide-react";
import { useMemo } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Spinner } from "@/components/ui/spinner";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useContainerStats } from "@/features/containers/hooks/use-container-stats";
import type { ContainerInfo } from "@/features/containers/types";

interface ContainerStatsCardProps {
  container: ContainerInfo;
  isEnabled: boolean;
  onToggle: () => void;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / k ** i).toFixed(1)} ${sizes[i]}`;
}

function formatContainerName(names: string[]): string {
  if (!names || names.length === 0) return "Unknown";
  return names[0].replace(/^\//, "");
}

export function ContainerStatsCard({
  container,
  isEnabled,
  onToggle,
}: ContainerStatsCardProps) {
  const { stats, isConnected, error, connect, disconnect } = useContainerStats({
    containerId: container.id,
    host: container.host,
    enabled: isEnabled,
  });

  const handleToggle = () => {
    if (isConnected) {
      disconnect();
    } else {
      connect();
    }
    onToggle();
  };

  const cpuColor = useMemo(() => {
    if (!stats) return "bg-muted";
    if (stats.cpu_percent > 80) return "bg-red-500";
    if (stats.cpu_percent > 60) return "bg-yellow-500";
    return "bg-green-500";
  }, [stats]);

  const memoryColor = useMemo(() => {
    if (!stats) return "bg-muted";
    if (stats.memory_percent > 80) return "bg-red-500";
    if (stats.memory_percent > 60) return "bg-yellow-500";
    return "bg-green-500";
  }, [stats]);

  return (
    <Card className="flex flex-col">
      <CardHeader className="pb-2">
        <div className="flex items-start justify-between gap-2">
          <div className="flex-1 min-w-0">
            <CardTitle className="text-base truncate" title={formatContainerName(container.names)}>
              {formatContainerName(container.names)}
            </CardTitle>
            <p className="text-xs text-muted-foreground truncate" title={container.image}>
              {container.image}
            </p>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <Badge variant={isConnected ? "default" : "secondary"} className="text-xs">
              {isConnected ? "Live" : "Off"}
            </Badge>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant={isConnected ? "default" : "outline"}
                  size="icon-sm"
                  onClick={handleToggle}
                >
                  {isConnected ? (
                    <SquareIcon className="size-3" />
                  ) : (
                    <PlayIcon className="size-3" />
                  )}
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                {isConnected ? "Stop streaming" : "Start streaming"}
              </TooltipContent>
            </Tooltip>
          </div>
        </div>
      </CardHeader>
      <CardContent className="flex-1 pt-0">
        {error && (
          <p className="text-xs text-destructive mb-2">{error}</p>
        )}

        {isConnected && !stats && (
          <div className="flex items-center justify-center py-4 text-muted-foreground">
            <Spinner className="mr-2 size-3" />
            <span className="text-xs">Connecting...</span>
          </div>
        )}

        {!isConnected && !stats && (
          <div className="py-4 text-center text-muted-foreground text-xs">
            Click play to stream stats
          </div>
        )}

        {stats && (
          <div className="space-y-3">
            {/* CPU */}
            <div className="space-y-1">
              <div className="flex items-center justify-between text-xs">
                <div className="flex items-center gap-1.5">
                  <CpuIcon className="size-3 text-muted-foreground" />
                  <span>CPU</span>
                </div>
                <span className="font-medium">{stats.cpu_percent.toFixed(1)}%</span>
              </div>
              <Progress value={Math.min(stats.cpu_percent, 100)} className="h-1.5" indicatorClassName={cpuColor} />
            </div>

            {/* Memory */}
            <div className="space-y-1">
              <div className="flex items-center justify-between text-xs">
                <div className="flex items-center gap-1.5">
                  <MemoryStickIcon className="size-3 text-muted-foreground" />
                  <span>Memory</span>
                </div>
                <span className="font-medium">
                  {formatBytes(stats.memory_usage)} / {formatBytes(stats.memory_limit)}
                </span>
              </div>
              <Progress value={Math.min(stats.memory_percent, 100)} className="h-1.5" indicatorClassName={memoryColor} />
            </div>

            {/* Network I/O */}
            <div className="flex items-center justify-between text-xs">
              <div className="flex items-center gap-1.5">
                <NetworkIcon className="size-3 text-muted-foreground" />
                <span>Network</span>
              </div>
              <span className="font-mono text-[10px]">
                {formatBytes(stats.network_rx)} / {formatBytes(stats.network_tx)}
              </span>
            </div>

            {/* Block I/O */}
            <div className="flex items-center justify-between text-xs">
              <div className="flex items-center gap-1.5">
                <HardDriveIcon className="size-3 text-muted-foreground" />
                <span>Disk</span>
              </div>
              <span className="font-mono text-[10px]">
                {formatBytes(stats.block_read)} / {formatBytes(stats.block_write)}
              </span>
            </div>

            {/* PIDs */}
            <div className="flex items-center justify-between text-xs">
              <div className="flex items-center gap-1.5">
                <ActivityIcon className="size-3 text-muted-foreground" />
                <span>PIDs</span>
              </div>
              <span className="font-medium">{stats.pids}</span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
