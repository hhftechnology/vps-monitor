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
import { Card, CardContent } from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import { useContainerStats } from "../hooks/use-container-stats";

interface ContainerStatsProps {
  containerId: string;
  host: string;
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / k ** i).toFixed(1)} ${sizes[i]}`;
}

function formatPercent(value: number): string {
  return `${value.toFixed(1)}%`;
}

export function ContainerStats({ containerId, host }: ContainerStatsProps) {
  const { stats, isConnected, error, connect, disconnect } = useContainerStats({
    containerId,
    host,
    enabled: false, // Start disconnected, user can toggle
  });

  const statsCards = useMemo(() => {
    if (!stats) return null;

    return [
      {
        label: "CPU",
        value: formatPercent(stats.cpu_percent),
        icon: CpuIcon,
        color: stats.cpu_percent > 80 ? "text-red-500" : "text-primary",
      },
      {
        label: "Memory",
        value: `${formatBytes(stats.memory_usage)} / ${formatBytes(stats.memory_limit)}`,
        subValue: formatPercent(stats.memory_percent),
        icon: MemoryStickIcon,
        color: stats.memory_percent > 80 ? "text-red-500" : "text-primary",
      },
      {
        label: "Network I/O",
        value: `${formatBytes(stats.network_rx)} / ${formatBytes(stats.network_tx)}`,
        subLabel: "RX / TX",
        icon: NetworkIcon,
        color: "text-primary",
      },
      {
        label: "Block I/O",
        value: `${formatBytes(stats.block_read)} / ${formatBytes(stats.block_write)}`,
        subLabel: "Read / Write",
        icon: HardDriveIcon,
        color: "text-primary",
      },
      {
        label: "PIDs",
        value: stats.pids.toString(),
        icon: ActivityIcon,
        color: "text-primary",
      },
    ];
  }, [stats]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium">Real-time Stats</h3>
          <Badge variant={isConnected ? "default" : "secondary"}>
            {isConnected ? "Live" : "Disconnected"}
          </Badge>
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant={isConnected ? "default" : "outline"}
              size="sm"
              onClick={isConnected ? disconnect : connect}
            >
              {isConnected ? (
                <>
                  <SquareIcon className="mr-2 size-4" />
                  Stop
                </>
              ) : (
                <>
                  <PlayIcon className="mr-2 size-4" />
                  Start
                </>
              )}
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {isConnected ? "Stop streaming stats" : "Start streaming stats"}
          </TooltipContent>
        </Tooltip>
      </div>

      {error && (
        <p className="text-sm text-destructive">
          {error}
        </p>
      )}

      {isConnected && !stats && (
        <div className="flex items-center justify-center py-8 text-muted-foreground">
          <Spinner className="mr-2 size-4" />
          Connecting...
        </div>
      )}

      {!isConnected && !stats && (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground text-sm">
            Click "Start" to stream real-time container stats
          </CardContent>
        </Card>
      )}

      {statsCards && (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {statsCards.map((card) => {
            const Icon = card.icon;
            return (
              <Card key={card.label}>
                <CardContent className="p-4">
                  <div className="flex items-center gap-3">
                    <div className={`p-2 rounded-md bg-muted ${card.color}`}>
                      <Icon className="size-4" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-xs text-muted-foreground">
                        {card.label}
                      </p>
                      <p className="text-sm font-medium truncate">
                        {card.value}
                      </p>
                      {card.subValue && (
                        <p className="text-xs text-muted-foreground">
                          {card.subValue}
                        </p>
                      )}
                      {card.subLabel && (
                        <p className="text-xs text-muted-foreground">
                          {card.subLabel}
                        </p>
                      )}
                    </div>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
