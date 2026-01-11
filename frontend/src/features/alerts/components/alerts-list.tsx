import { useMemo, useState } from "react";
import {
  AlertTriangleIcon,
  BellIcon,
  CheckIcon,
  CpuIcon,
  MemoryStickIcon,
  RefreshCcwIcon,
  SquareIcon,
} from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import {
  useAcknowledgeAlertMutation,
  useAlertConfigQuery,
  useAlertsQuery,
} from "../hooks/use-alerts-query";
import type { Alert, AlertType } from "../types";

function getAlertIcon(type: AlertType) {
  switch (type) {
    case "container_stopped":
      return SquareIcon;
    case "cpu_threshold":
      return CpuIcon;
    case "memory_threshold":
      return MemoryStickIcon;
    default:
      return AlertTriangleIcon;
  }
}

function getAlertColor(type: AlertType, acknowledged: boolean) {
  if (acknowledged) return "text-muted-foreground";
  switch (type) {
    case "container_stopped":
      return "text-yellow-500";
    case "cpu_threshold":
    case "memory_threshold":
      return "text-red-500";
    default:
      return "text-orange-500";
  }
}

function formatAlertType(type: AlertType): string {
  switch (type) {
    case "container_stopped":
      return "Container Stopped";
    case "cpu_threshold":
      return "CPU Threshold";
    case "memory_threshold":
      return "Memory Threshold";
    default:
      return type;
  }
}

function formatTimestamp(timestamp: number): string {
  const date = new Date(timestamp * 1000);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  if (diff < 60000) {
    return "Just now";
  } else if (diff < 3600000) {
    const mins = Math.floor(diff / 60000);
    return `${mins}m ago`;
  } else if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000);
    return `${hours}h ago`;
  } else {
    return date.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  }
}

export function AlertsList() {
  const { data, isLoading, error, refetch, isRefetching } = useAlertsQuery();
  const { data: config } = useAlertConfigQuery();
  const acknowledgeMutation = useAcknowledgeAlertMutation();

  const [showAcknowledged, setShowAcknowledged] = useState(false);

  const filteredAlerts = useMemo(() => {
    if (!data?.alerts) return [];
    if (showAcknowledged) return data.alerts;
    return data.alerts.filter((alert) => !alert.acknowledged);
  }, [data?.alerts, showAcknowledged]);

  const unacknowledgedCount = useMemo(() => {
    if (!data?.alerts) return 0;
    return data.alerts.filter((alert) => !alert.acknowledged).length;
  }, [data?.alerts]);

  const handleAcknowledge = async (alert: Alert) => {
    try {
      await acknowledgeMutation.mutateAsync(alert.id);
      toast.success("Alert acknowledged");
    } catch (err) {
      toast.error(
        `Failed to acknowledge alert: ${err instanceof Error ? err.message : "Unknown error"}`
      );
    }
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Spinner className="mr-2 size-4" />
          Loading alerts...
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-destructive">
          Failed to load alerts: {error.message}
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {config && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Alert Configuration</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              <div className="flex items-center gap-2">
                <Badge variant={config.enabled ? "default" : "secondary"}>
                  {config.enabled ? "Enabled" : "Disabled"}
                </Badge>
              </div>
              <div className="text-sm">
                <span className="text-muted-foreground">CPU Threshold: </span>
                <span className="font-medium">{config.cpu_threshold}%</span>
              </div>
              <div className="text-sm">
                <span className="text-muted-foreground">Memory Threshold: </span>
                <span className="font-medium">{config.memory_threshold}%</span>
              </div>
              <div className="text-sm">
                <span className="text-muted-foreground">Check Interval: </span>
                <span className="font-medium">{config.check_interval}</span>
              </div>
            </div>
            {config.webhook_configured && (
              <p className="mt-3 text-sm text-muted-foreground">
                Webhook notifications are configured
              </p>
            )}
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <CardTitle>Alerts</CardTitle>
              {unacknowledgedCount > 0 && (
                <Badge variant="destructive">{unacknowledgedCount} new</Badge>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant={showAcknowledged ? "secondary" : "outline"}
                size="sm"
                onClick={() => setShowAcknowledged(!showAcknowledged)}
              >
                {showAcknowledged ? "Hide" : "Show"} acknowledged
              </Button>
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
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {filteredAlerts.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <BellIcon className="size-12 mb-4 opacity-50" />
              <p className="text-lg font-medium">No alerts</p>
              <p className="text-sm">
                {showAcknowledged
                  ? "No alerts have been triggered"
                  : "All alerts have been acknowledged"}
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {filteredAlerts.map((alert) => {
                const Icon = getAlertIcon(alert.type);
                const color = getAlertColor(alert.type, alert.acknowledged);

                return (
                  <div
                    key={alert.id}
                    className={`flex items-start gap-4 p-4 rounded-lg border ${
                      alert.acknowledged
                        ? "bg-muted/30 opacity-60"
                        : "bg-card"
                    }`}
                  >
                    <div className={`mt-0.5 ${color}`}>
                      <Icon className="size-5" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <Badge
                          variant={alert.acknowledged ? "outline" : "secondary"}
                        >
                          {formatAlertType(alert.type)}
                        </Badge>
                        <span className="text-xs text-muted-foreground">
                          {formatTimestamp(alert.timestamp)}
                        </span>
                        {alert.acknowledged && (
                          <Badge variant="outline" className="text-xs">
                            Acknowledged
                          </Badge>
                        )}
                      </div>
                      <p className="text-sm font-medium">{alert.message}</p>
                      <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                        <span>Container: {alert.container_name}</span>
                        <span>Host: {alert.host}</span>
                        {alert.value !== undefined &&
                          alert.threshold !== undefined && (
                            <span>
                              Value: {alert.value.toFixed(1)}% (threshold:{" "}
                              {alert.threshold}%)
                            </span>
                          )}
                      </div>
                    </div>
                    {!alert.acknowledged && (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => handleAcknowledge(alert)}
                            disabled={acknowledgeMutation.isPending}
                          >
                            {acknowledgeMutation.isPending ? (
                              <Spinner className="size-4" />
                            ) : (
                              <CheckIcon className="size-4" />
                            )}
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>Acknowledge</TooltipContent>
                      </Tooltip>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
