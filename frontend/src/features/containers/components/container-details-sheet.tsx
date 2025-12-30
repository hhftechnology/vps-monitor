import {
  ActivityIcon,
  CpuIcon,
  HardDriveIcon,
  MemoryStickIcon,
  NetworkIcon,
  PlayIcon,
  SettingsIcon,
  SquareIcon,
  TerminalIcon,
} from "lucide-react";
import { lazy, Suspense, useCallback, useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import { useContainerStats } from "../hooks/use-container-stats";
import { ContainerStatsCharts } from "./container-stats-charts";
import { EnvironmentVariables } from "./environment-variables";

import type { ContainerInfo } from "../types";

// Lazy load terminal to reduce initial bundle size
const Terminal = lazy(() =>
  import("./terminal").then((module) => ({ default: module.Terminal }))
);

interface ContainerDetailsSheetProps {
  container: ContainerInfo | null;
  host: string;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  isReadOnly?: boolean;
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

export function ContainerDetailsSheet({
  container,
  host,
  isOpen,
  onOpenChange,
  isReadOnly = false,
}: ContainerDetailsSheetProps) {
  const [activeTab, setActiveTab] = useState("stats");
  const [containerId, setContainerId] = useState(container?.id ?? "");

  // Update containerId when container changes
  const effectiveContainerId = container?.id ?? containerId;

  const {
    stats,
    history,
    isConnected,
    error,
    connect,
    disconnect,
    clearHistory,
  } = useContainerStats({
    containerId: effectiveContainerId,
    host,
    enabled: isOpen && activeTab === "stats",
  });

  const handleContainerIdChange = useCallback((newId: string) => {
    setContainerId(newId);
  }, []);

  const handleToggleStats = useCallback(() => {
    if (isConnected) {
      disconnect();
    } else {
      clearHistory();
      connect();
    }
  }, [isConnected, disconnect, clearHistory, connect]);

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

  if (!container) return null;

  const containerName = container.names?.[0]?.replace(/^\//, "") || container.id.slice(0, 12);
  const isRunning = container.state.toLowerCase() === "running";

  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-2xl w-full overflow-y-auto">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            {containerName}
            <Badge
              variant={isRunning ? "default" : "secondary"}
              className="text-xs"
            >
              {container.state}
            </Badge>
          </SheetTitle>
          <SheetDescription className="text-xs font-mono">
            {container.image}
          </SheetDescription>
        </SheetHeader>

        <Tabs
          value={activeTab}
          onValueChange={setActiveTab}
          className="mt-6"
        >
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="stats" className="flex items-center gap-2">
              <ActivityIcon className="size-4" />
              Stats
            </TabsTrigger>
            <TabsTrigger
              value="terminal"
              className="flex items-center gap-2"
              disabled={!isRunning}
            >
              <TerminalIcon className="size-4" />
              Terminal
            </TabsTrigger>
            <TabsTrigger value="env" className="flex items-center gap-2">
              <SettingsIcon className="size-4" />
              Env Vars
            </TabsTrigger>
          </TabsList>

          <TabsContent value="stats" className="space-y-4">
            {/* Stats Controls */}
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Badge variant={isConnected ? "default" : "secondary"}>
                  {isConnected ? "Live" : "Disconnected"}
                </Badge>
                {history.length > 0 && (
                  <span className="text-xs text-muted-foreground">
                    {history.length} data points
                  </span>
                )}
              </div>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant={isConnected ? "default" : "outline"}
                    size="sm"
                    onClick={handleToggleStats}
                    disabled={!isRunning}
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
                  {!isRunning
                    ? "Container must be running"
                    : isConnected
                      ? "Stop streaming stats"
                      : "Start streaming stats"}
                </TooltipContent>
              </Tooltip>
            </div>

            {error && (
              <p className="text-sm text-destructive">{error}</p>
            )}

            {!isRunning && (
              <Card>
                <CardContent className="py-8 text-center text-muted-foreground text-sm">
                  Container is not running. Start the container to view stats.
                </CardContent>
              </Card>
            )}

            {isRunning && isConnected && !stats && (
              <div className="flex items-center justify-center py-8 text-muted-foreground">
                <Spinner className="mr-2 size-4" />
                Connecting...
              </div>
            )}

            {isRunning && !isConnected && !stats && (
              <Card>
                <CardContent className="py-8 text-center text-muted-foreground text-sm">
                  Click "Start" to stream real-time container stats
                </CardContent>
              </Card>
            )}

            {/* Live Stats Cards */}
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

            {/* Stats Charts */}
            <ContainerStatsCharts history={history} />
          </TabsContent>

          <TabsContent value="terminal" className="min-h-[400px]">
            {isRunning ? (
              <Suspense
                fallback={
                  <div className="flex items-center justify-center h-[400px] text-muted-foreground">
                    <Spinner className="mr-2 size-4" />
                    Loading terminal...
                  </div>
                }
              >
                <Terminal containerId={effectiveContainerId} host={host} />
              </Suspense>
            ) : (
              <Card>
                <CardContent className="py-8 text-center text-muted-foreground text-sm">
                  Container is not running. Start the container to access terminal.
                </CardContent>
              </Card>
            )}
          </TabsContent>

          <TabsContent value="env">
            <EnvironmentVariables
              containerId={effectiveContainerId}
              containerHost={host}
              isReadOnly={isReadOnly}
              onContainerIdChange={handleContainerIdChange}
            />
          </TabsContent>
        </Tabs>
      </SheetContent>
    </Sheet>
  );
}
