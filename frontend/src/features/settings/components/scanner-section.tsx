import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import isEqual from "fast-deep-equal";

import {
  useScannerConfig,
  useTestScanNotification,
  useUpdateScannerConfig,
} from "@/features/scanner/hooks/use-scan-query";
import type { ScannerConfig, SeverityLevel } from "@/features/scanner/types";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";

interface ScannerSectionProps {
  disabled?: boolean;
}

const severityOptions: SeverityLevel[] = [
  "Critical",
  "High",
  "Medium",
  "Low",
  "Negligible",
  "Unknown",
];

function configsMatch(a: ScannerConfig | null, b: ScannerConfig | null) {
  return isEqual(a, b);
}

export function ScannerSection({ disabled = false }: ScannerSectionProps) {
  const { data, isLoading, error } = useScannerConfig();
  const updateMutation = useUpdateScannerConfig();
  const testMutation = useTestScanNotification();
  const [draft, setDraft] = useState<ScannerConfig | null>(null);

  useEffect(() => {
    if (data) {
      setDraft(data);
    }
  }, [data]);

  const hasChanges = useMemo(() => {
    if (!data || !draft) return false;
    return !configsMatch(data, draft);
  }, [data, draft]);

  const saveConfig = async () => {
    if (!draft) return null;
    const updated = await updateMutation.mutateAsync(draft);
    toast.success("Scanner configuration saved");
    setDraft(updated);
    return updated;
  };

  const handleSave = async () => {
    try {
      await saveConfig();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to save scanner configuration");
    }
  };

  const handleTest = async () => {
    try {
      if (hasChanges) {
        await saveConfig();
      }
      await testMutation.mutateAsync();
      toast.success("Test notification sent");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Failed to send test notification");
    }
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Scanner</CardTitle>
          <CardDescription>Loading scanner configuration...</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  if (error || !draft) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Scanner</CardTitle>
          <CardDescription>
            Failed to load scanner settings: {error?.message ?? "Unknown error"}
          </CardDescription>
        </CardHeader>
      </Card>
    );
  }

  const busy = disabled || updateMutation.isPending || testMutation.isPending;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Scanner</CardTitle>
        <CardDescription>
          Configure vulnerability scanning, SBOM generation, and completion notifications.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="grype-image">Grype image</Label>
            <Input
              id="grype-image"
              value={draft.grypeImage}
              onChange={(e) => setDraft({ ...draft, grypeImage: e.target.value })}
              disabled={busy}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="trivy-image">Trivy image</Label>
            <Input
              id="trivy-image"
              value={draft.trivyImage}
              onChange={(e) => setDraft({ ...draft, trivyImage: e.target.value })}
              disabled={busy}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="syft-image">Syft image</Label>
            <Input
              id="syft-image"
              value={draft.syftImage}
              onChange={(e) => setDraft({ ...draft, syftImage: e.target.value })}
              disabled={busy}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="default-scanner">Default scanner</Label>
            <Select
              value={draft.defaultScanner}
              onValueChange={(value) =>
                setDraft({ ...draft, defaultScanner: value as ScannerConfig["defaultScanner"] })
              }
              disabled={busy}
            >
              <SelectTrigger id="default-scanner">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="grype">Grype</SelectItem>
                <SelectItem value="trivy">Trivy</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="grype-args">Grype args</Label>
            <Input
              id="grype-args"
              value={draft.grypeArgs}
              onChange={(e) => setDraft({ ...draft, grypeArgs: e.target.value })}
              disabled={busy}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="trivy-args">Trivy args</Label>
            <Input
              id="trivy-args"
              value={draft.trivyArgs}
              onChange={(e) => setDraft({ ...draft, trivyArgs: e.target.value })}
              disabled={busy}
            />
          </div>
        </div>

        <div className="space-y-4 rounded-lg border p-4">
          <div>
            <h3 className="font-medium">Notifications</h3>
            <p className="text-sm text-muted-foreground">
              Send scanner completion updates to Discord and Slack webhooks.
            </p>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="discord-webhook">Discord webhook URL</Label>
              <Input
                id="discord-webhook"
                value={draft.notifications.discordWebhookURL ?? ""}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    notifications: {
                      ...draft.notifications,
                      discordWebhookURL: e.target.value,
                    },
                  })
                }
                disabled={busy}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="slack-webhook">Slack webhook URL</Label>
              <Input
                id="slack-webhook"
                value={draft.notifications.slackWebhookURL ?? ""}
                onChange={(e) =>
                  setDraft({
                    ...draft,
                    notifications: {
                      ...draft.notifications,
                      slackWebhookURL: e.target.value,
                    },
                  })
                }
                disabled={busy}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="min-severity">Minimum severity</Label>
              <Select
                value={draft.notifications.minSeverity || "High"}
                onValueChange={(value) =>
                  setDraft({
                    ...draft,
                    notifications: {
                      ...draft.notifications,
                      minSeverity: value as SeverityLevel,
                    },
                  })
                }
                disabled={busy}
              >
                <SelectTrigger id="min-severity">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {severityOptions.map((severity) => (
                    <SelectItem key={severity} value={severity}>
                      {severity}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div className="flex items-center justify-between rounded-md border p-3">
              <div>
                <p className="text-sm font-medium">On scan complete</p>
                <p className="text-xs text-muted-foreground">
                  Send a notification when a single image scan finishes.
                </p>
              </div>
              <Switch
                checked={draft.notifications.onScanComplete}
                onCheckedChange={(value) =>
                  setDraft({
                    ...draft,
                    notifications: {
                      ...draft.notifications,
                      onScanComplete: value,
                    },
                  })
                }
                disabled={busy}
              />
            </div>
            <div className="flex items-center justify-between rounded-md border p-3">
              <div>
                <p className="text-sm font-medium">On bulk complete</p>
                <p className="text-xs text-muted-foreground">
                  Send a notification when the bulk scan finishes.
                </p>
              </div>
              <Switch
                checked={draft.notifications.onBulkComplete}
                onCheckedChange={(value) =>
                  setDraft({
                    ...draft,
                    notifications: {
                      ...draft.notifications,
                      onBulkComplete: value,
                    },
                  })
                }
                disabled={busy}
              />
            </div>
          </div>
        </div>

        <div className="space-y-4 rounded-lg border p-4">
          <div>
            <h3 className="font-medium">Resource limits</h3>
            <p className="text-sm text-muted-foreground">
              Tune timeouts and resource ceilings for spawned scanner containers. Increase these
              for very large images or slow networks.
            </p>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="scan-timeout">Scan timeout (minutes)</Label>
              <Input
                id="scan-timeout"
                type="number"
                min={1}
                value={draft.scanTimeoutMinutes ?? 20}
                onChange={(e) =>
                  setDraft({ ...draft, scanTimeoutMinutes: Number(e.target.value) || 0 })
                }
                disabled={busy}
              />
              <p className="text-xs text-muted-foreground">Per single-image scan. Default: 20.</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="bulk-timeout">Bulk timeout (minutes)</Label>
              <Input
                id="bulk-timeout"
                type="number"
                min={1}
                value={draft.bulkTimeoutMinutes ?? 120}
                onChange={(e) =>
                  setDraft({ ...draft, bulkTimeoutMinutes: Number(e.target.value) || 0 })
                }
                disabled={busy}
              />
              <p className="text-xs text-muted-foreground">For full host bulk scans. Default: 120.</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="scanner-memory">Scanner memory (MB)</Label>
              <Input
                id="scanner-memory"
                type="number"
                min={128}
                value={draft.scannerMemoryMB ?? 2048}
                onChange={(e) =>
                  setDraft({ ...draft, scannerMemoryMB: Number(e.target.value) || 0 })
                }
                disabled={busy}
              />
              <p className="text-xs text-muted-foreground">
                Memory ceiling per scanner container. Default: 2048.
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="scanner-pids">PID limit</Label>
              <Input
                id="scanner-pids"
                type="number"
                min={32}
                value={draft.scannerPidsLimit ?? 512}
                onChange={(e) =>
                  setDraft({ ...draft, scannerPidsLimit: Number(e.target.value) || 0 })
                }
                disabled={busy}
              />
              <p className="text-xs text-muted-foreground">
                Max processes per scanner container. Default: 512.
              </p>
            </div>
          </div>
        </div>

        {disabled && (
          <p className="text-sm text-muted-foreground">
            Scanner settings are disabled while the server is in read-only mode.
          </p>
        )}

        <div className="flex flex-wrap justify-end gap-2">
          <Button variant="outline" onClick={handleTest} disabled={busy}>
            {testMutation.isPending ? "Testing..." : "Test Notification"}
          </Button>
          <Button onClick={handleSave} disabled={busy || !hasChanges}>
            {updateMutation.isPending ? "Saving..." : "Save Scanner Settings"}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
