import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";

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
  useTestCoolifyHost,
  useUpdateCoolifyHosts,
} from "../hooks/use-settings";
import type { CoolifyHostsConfig } from "../types";
import { EnvBadge } from "./env-badge";

interface CoolifyHostsSectionProps {
  config: CoolifyHostsConfig;
}

interface EditingHost {
  hostName: string;
  apiURL: string;
  apiToken: string;
}

const EMPTY_HOST: EditingHost = { hostName: "", apiURL: "", apiToken: "" };
const MASKED_TOKEN = "••••••••";

export function CoolifyHostsSection({ config }: CoolifyHostsSectionProps) {
  const envHosts = config.hosts.filter((h) => h.source === "env");
  const [fileHosts, setFileHosts] = useState<EditingHost[]>(
    config.hosts
      .filter((h) => h.source !== "env")
      .map(({ hostName, apiURL, apiToken }) => ({ hostName, apiURL, apiToken })),
  );
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editingHost, setEditingHost] = useState<EditingHost>(EMPTY_HOST);
  const [isAdding, setIsAdding] = useState(false);
  const [newHost, setNewHost] = useState<EditingHost>(EMPTY_HOST);
  const [testResults, setTestResults] = useState<
    Record<string, { success: boolean; message: string }>
  >({});
  const [testingKeys, setTestingKeys] = useState<Set<string>>(new Set());

  const updateMutation = useUpdateCoolifyHosts();
  const testMutation = useTestCoolifyHost();

  const originalFileHosts = useMemo(
    () => config.hosts.filter((h) => h.source !== "env").map(({ hostName, apiURL, apiToken }) => ({ hostName, apiURL, apiToken })),
    [config.hosts],
  );

  useEffect(() => {
    setFileHosts(
      config.hosts
        .filter((h) => h.source !== "env")
        .map(({ hostName, apiURL, apiToken }) => ({ hostName, apiURL, apiToken })),
    );
    setEditingIndex(null);
    setEditingHost(EMPTY_HOST);
    setIsAdding(false);
    setNewHost(EMPTY_HOST);
    setTestResults({});
    setTestingKeys(new Set());
  }, [config.hosts]);

  const hasChanges = fileHosts.length !== originalFileHosts.length ||
    fileHosts.some((h, i) => h.hostName !== originalFileHosts[i]?.hostName || h.apiURL !== originalFileHosts[i]?.apiURL || h.apiToken !== originalFileHosts[i]?.apiToken);

  function handleSave() {
    updateMutation.mutate(fileHosts, {
      onSuccess: (msg) => toast.success(msg),
      onError: (err) => toast.error(err.message),
    });
  }

  function handleRemove(index: number) {
    setFileHosts((prev) => prev.filter((_, i) => i !== index));
    setTestResults({});
  }

  function handleStartEdit(index: number) {
    setEditingIndex(index);
    setEditingHost({ ...fileHosts[index] });
  }

  function handleCancelEdit() {
    setEditingIndex(null);
    setEditingHost(EMPTY_HOST);
  }

  function handleSaveEdit() {
    if (editingIndex === null) return;
    if (!editingHost.hostName.trim() || !editingHost.apiURL.trim() || !editingHost.apiToken.trim()) {
      toast.error("Host name, API URL, and API token are all required");
      return;
    }
    const trimmedName = editingHost.hostName.trim();
    if (envHosts.some((h) => h.hostName === trimmedName)) {
      toast.error(`Host name "${trimmedName}" is defined via environment variable`);
      return;
    }
    if (fileHosts.some((h, i) => i !== editingIndex && h.hostName === trimmedName)) {
      toast.error(`Host name "${trimmedName}" already exists`);
      return;
    }
    const next = [...fileHosts];
    next[editingIndex] = {
      hostName: trimmedName,
      apiURL: editingHost.apiURL.trim(),
      apiToken: editingHost.apiToken.trim(),
    };
    setFileHosts(next);
    setEditingIndex(null);
    setEditingHost(EMPTY_HOST);
    setTestResults({});
  }

  function handleAddHost() {
    if (!newHost.hostName.trim() || !newHost.apiURL.trim() || !newHost.apiToken.trim()) {
      toast.error("Host name, API URL, and API token are all required");
      return;
    }
    const trimmedName = newHost.hostName.trim();
    if (envHosts.some((h) => h.hostName === trimmedName)) {
      toast.error(`Host name "${trimmedName}" is already defined via environment variable`);
      return;
    }
    if (fileHosts.some((h) => h.hostName === trimmedName)) {
      toast.error(`Host name "${trimmedName}" already exists`);
      return;
    }
    setFileHosts([...fileHosts, { hostName: trimmedName, apiURL: newHost.apiURL.trim(), apiToken: newHost.apiToken.trim() }]);
    setNewHost(EMPTY_HOST);
    setIsAdding(false);
    setTestResults({});
  }

  function handleTest(key: string, h: { hostName: string; apiURL: string; apiToken: string }) {
    setTestingKeys((prev) => {
      const next = new Set(prev);
      next.add(key);
      return next;
    });
    testMutation.mutate(
      { hostName: h.hostName, apiURL: h.apiURL, apiToken: h.apiToken },
      {
        onSuccess: (result) => {
          setTestResults((prev) => ({
            ...prev,
            [key]: { success: result.success, message: result.message },
          }));
          setTestingKeys((prev) => {
            const next = new Set(prev);
            next.delete(key);
            return next;
          });
        },
        onError: (err) => {
          setTestResults((prev) => ({
            ...prev,
            [key]: { success: false, message: err.message },
          }));
          setTestingKeys((prev) => {
            const next = new Set(prev);
            next.delete(key);
            return next;
          });
        },
      },
    );
  }

  function renderRow(
    h: { hostName: string; apiURL: string; apiToken: string },
    isEnvRow: boolean,
    fileIndex?: number,
  ) {
    const testKey = `${isEnvRow ? "env" : "file"}-${h.hostName}`;
    const isEditing = !isEnvRow && editingIndex === fileIndex;

    if (isEditing && fileIndex !== undefined) {
      return (
        <TableRow key={testKey}>
          <TableCell>
            <Input
              value={editingHost.hostName}
              onChange={(e) => setEditingHost((prev) => ({ ...prev, hostName: e.target.value }))}
              className="h-8"
            />
          </TableCell>
          <TableCell>
            <Input
              value={editingHost.apiURL}
              onChange={(e) => setEditingHost((prev) => ({ ...prev, apiURL: e.target.value }))}
              className="h-8"
              placeholder="https://coolify.example.com"
            />
          </TableCell>
          <TableCell>
            <Input
              value={editingHost.apiToken}
              onChange={(e) => setEditingHost((prev) => ({ ...prev, apiToken: e.target.value }))}
              className="h-8"
              placeholder="Leave masked to keep existing"
              type="password"
            />
          </TableCell>
          <TableCell />
          <TableCell className="text-right">
            <div className="flex items-center justify-end gap-1">
              <Button variant="ghost" size="sm" onClick={handleSaveEdit}>Done</Button>
              <Button variant="ghost" size="sm" onClick={handleCancelEdit}>Cancel</Button>
            </div>
          </TableCell>
        </TableRow>
      );
    }

    return (
      <TableRow key={testKey}>
        <TableCell className="font-medium">
          <div className="flex items-center gap-2">
            {h.hostName}
            {isEnvRow && <EnvBadge />}
          </div>
        </TableCell>
        <TableCell className="font-mono text-xs text-muted-foreground">{h.apiURL}</TableCell>
        <TableCell className="text-muted-foreground">{MASKED_TOKEN}</TableCell>
        <TableCell>
          {testResults[testKey] && (
            <span
              className={`text-xs ${testResults[testKey].success ? "text-green-600 dark:text-green-400" : "text-red-600 dark:text-red-400"}`}
            >
              {testResults[testKey].message}
            </span>
          )}
        </TableCell>
        <TableCell className="text-right">
          <div className="flex items-center justify-end gap-1">
            {!isEnvRow && (
              <Button
                variant="ghost"
                size="sm"
                disabled={testingKeys.has(testKey)}
                onClick={() => handleTest(testKey, h)}
              >
                {testingKeys.has(testKey) ? <Spinner className="size-3" /> : "Test"}
              </Button>
            )}
            {!isEnvRow && fileIndex !== undefined && (
              <>
                <Button variant="ghost" size="sm" disabled={editingIndex !== null} onClick={() => handleStartEdit(fileIndex)}>Edit</Button>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={editingIndex !== null}
                  onClick={() => handleRemove(fileIndex)}
                  className="text-destructive hover:text-destructive"
                >
                  Remove
                </Button>
              </>
            )}
          </div>
        </TableCell>
      </TableRow>
    );
  }

  const allHosts = [
    ...envHosts.map((h) => ({ ...h, isEnv: true as const })),
    ...fileHosts.map((h, i) => ({ ...h, source: "file" as const, isEnv: false as const, fileIndex: i })),
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Coolify Hosts</CardTitle>
        <CardDescription>
          Connect to Coolify instances to persist environment variable changes across redeployments.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {allHosts.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>API URL</TableHead>
                <TableHead>Token</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {allHosts.map((h) =>
                renderRow(
                  { hostName: h.hostName, apiURL: h.apiURL, apiToken: h.apiToken },
                  h.isEnv,
                  h.isEnv ? undefined : h.fileIndex,
                ),
              )}
            </TableBody>
          </Table>
        )}

        {allHosts.length === 0 && !isAdding && (
          <p className="text-sm text-muted-foreground">No Coolify hosts configured.</p>
        )}

        {isAdding && (
          <div className="flex items-end gap-3 border rounded-md p-3">
            <div className="space-y-1.5 flex-1">
              <Label htmlFor="new-coolify-name">Name</Label>
              <Input
                id="new-coolify-name"
                value={newHost.hostName}
                onChange={(e) => setNewHost((prev) => ({ ...prev, hostName: e.target.value }))}
                placeholder="production"
                className="h-8"
              />
            </div>
            <div className="space-y-1.5 flex-1">
              <Label htmlFor="new-coolify-url">API URL</Label>
              <Input
                id="new-coolify-url"
                value={newHost.apiURL}
                onChange={(e) => setNewHost((prev) => ({ ...prev, apiURL: e.target.value }))}
                placeholder="https://coolify.example.com"
                className="h-8"
              />
            </div>
            <div className="space-y-1.5 flex-1">
              <Label htmlFor="new-coolify-token">API Token</Label>
              <Input
                id="new-coolify-token"
                value={newHost.apiToken}
                onChange={(e) => setNewHost((prev) => ({ ...prev, apiToken: e.target.value }))}
                placeholder="Token"
                type="password"
                className="h-8"
              />
            </div>
            <div className="flex gap-1">
              <Button size="sm" onClick={handleAddHost}>Add</Button>
              <Button size="sm" variant="ghost" onClick={() => { setIsAdding(false); setNewHost(EMPTY_HOST); }}>
                Cancel
              </Button>
            </div>
          </div>
        )}

        <div className="flex items-center gap-2">
          {!isAdding && (
            <Button variant="outline" size="sm" onClick={() => setIsAdding(true)}>
              Add host
            </Button>
          )}
          {hasChanges && (
            <Button size="sm" disabled={updateMutation.isPending} onClick={handleSave}>
              {updateMutation.isPending ? (
                <><Spinner className="size-3" /> Saving...</>
              ) : (
                "Save changes"
              )}
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
