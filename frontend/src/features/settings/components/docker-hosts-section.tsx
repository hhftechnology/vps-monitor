import { useMemo, useState } from "react";
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

import { useTestDockerHost, useUpdateDockerHosts } from "../hooks/use-settings";
import type { DockerHostsConfig } from "../types";
import { EnvBadge } from "./env-badge";

interface DockerHostsSectionProps {
  config: DockerHostsConfig;
}

interface EditingHost {
  name: string;
  host: string;
}

export function DockerHostsSection({ config }: DockerHostsSectionProps) {
  const envHosts = config.hosts.filter((h) => h.source === "env");
  const [fileHosts, setFileHosts] = useState<EditingHost[]>(
    config.hosts.filter((h) => h.source !== "env").map(({ name, host }) => ({ name, host })),
  );
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editingHost, setEditingHost] = useState<EditingHost>({
    name: "",
    host: "",
  });
  const [isAdding, setIsAdding] = useState(false);
  const [newHost, setNewHost] = useState<EditingHost>({
    name: "",
    host: "",
  });
  const [testResults, setTestResults] = useState<
    Record<string, { success: boolean; message: string }>
  >({});
  const [testingKeys, setTestingKeys] = useState<Set<string>>(new Set());

  const updateMutation = useUpdateDockerHosts();
  const testMutation = useTestDockerHost();

  const originalFileHosts = useMemo(
    () => config.hosts.filter((h) => h.source !== "env").map(({ name, host }) => ({ name, host })),
    [config.hosts],
  );
  const hasChanges = fileHosts.length !== originalFileHosts.length ||
    fileHosts.some((h, i) => h.name !== originalFileHosts[i]?.name || h.host !== originalFileHosts[i]?.host);

  function handleSave() {
    updateMutation.mutate(
      fileHosts.map(({ name, host }) => ({ name, host })),
      {
        onSuccess: (msg) => toast.success(msg),
        onError: (err) => toast.error(err.message),
      },
    );
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
    setEditingHost({ name: "", host: "" });
  }

  function handleSaveEdit() {
    if (editingIndex === null) return;
    if (!editingHost.name.trim() || !editingHost.host.trim()) {
      toast.error("Name and host are required");
      return;
    }
    const trimmedName = editingHost.name.trim();
    if (envHosts.some((h) => h.name === trimmedName)) {
      toast.error(`Host name "${trimmedName}" is defined via environment variable`);
      return;
    }
    if (fileHosts.some((h, i) => i !== editingIndex && h.name === trimmedName)) {
      toast.error(`Host name "${trimmedName}" already exists`);
      return;
    }
    const next = [...fileHosts];
    next[editingIndex] = { ...editingHost };
    setFileHosts(next);
    setEditingIndex(null);
    setEditingHost({ name: "", host: "" });
    setTestResults({});
  }

  function handleAddHost() {
    if (!newHost.name.trim() || !newHost.host.trim()) {
      toast.error("Name and host are required");
      return;
    }
    const trimmedName = newHost.name.trim();
    if (envHosts.some((h) => h.name === trimmedName)) {
      toast.error(`Host name "${trimmedName}" is already defined via environment variable`);
      return;
    }
    if (fileHosts.some((h) => h.name === trimmedName)) {
      toast.error(`Host name "${trimmedName}" already exists`);
      return;
    }
    setFileHosts([...fileHosts, { name: trimmedName, host: newHost.host.trim() }]);
    setNewHost({ name: "", host: "" });
    setIsAdding(false);
    setTestResults({});
  }

  function handleTest(key: string, hostUrl: string) {
    setTestingKeys((prev) => {
      const next = new Set(prev);
      next.add(key);
      return next;
    });
    setTestResults((prev) => {
      const next = { ...prev };
      delete next[key];
      return next;
    });
    testMutation.mutate(
      { name: key, host: hostUrl },
      {
        onSuccess: (result) => {
          setTestResults((prev) => ({
            ...prev,
            [key]: {
              success: result.success,
              message: result.success
                ? `Connected (Docker ${result.dockerVersion ?? ""})`
                : result.message,
            },
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
    h: { name: string; host: string },
    isEnvRow: boolean,
    fileIndex?: number,
  ) {
    const testKey = `${isEnvRow ? "env" : "file"}-${h.name}`;
    const isEditing = !isEnvRow && editingIndex === fileIndex;

    if (isEditing && fileIndex !== undefined) {
      return (
        <TableRow key={testKey}>
          <TableCell>
            <Input
              value={editingHost.name}
              onChange={(e) =>
                setEditingHost((prev) => ({ ...prev, name: e.target.value }))
              }
              className="h-8"
            />
          </TableCell>
          <TableCell>
            <Input
              value={editingHost.host}
              onChange={(e) =>
                setEditingHost((prev) => ({ ...prev, host: e.target.value }))
              }
              className="h-8"
              placeholder="unix:///var/run/docker.sock"
            />
          </TableCell>
          <TableCell />
          <TableCell className="text-right">
            <div className="flex items-center justify-end gap-1">
              <Button variant="ghost" size="sm" onClick={handleSaveEdit}>
                Done
              </Button>
              <Button variant="ghost" size="sm" onClick={handleCancelEdit}>
                Cancel
              </Button>
            </div>
          </TableCell>
        </TableRow>
      );
    }

    return (
      <TableRow key={testKey}>
        <TableCell className="font-medium">
          <div className="flex items-center gap-2">
            {h.name}
            {isEnvRow && <EnvBadge />}
          </div>
        </TableCell>
        <TableCell className="font-mono text-xs text-muted-foreground">
          {h.host}
        </TableCell>
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
            <Button
              variant="ghost"
              size="sm"
              disabled={testingKeys.has(testKey)}
              onClick={() => handleTest(testKey, h.host)}
            >
              {testingKeys.has(testKey) ? <Spinner className="size-3" /> : "Test"}
            </Button>
            {!isEnvRow && fileIndex !== undefined && (
              <>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={editingIndex !== null}
                  onClick={() => handleStartEdit(fileIndex)}
                >
                  Edit
                </Button>
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
        <CardTitle>Docker Hosts</CardTitle>
        <CardDescription>
          Configure which Docker daemon sockets or remote hosts to connect to.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {allHosts.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Host</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {allHosts.map((h) =>
                renderRow(
                  { name: h.name, host: h.host },
                  h.isEnv,
                  h.isEnv ? undefined : h.fileIndex,
                ),
              )}
            </TableBody>
          </Table>
        )}

        {allHosts.length === 0 && !isAdding && (
          <p className="text-sm text-muted-foreground">
            No Docker hosts configured.
          </p>
        )}

        {isAdding && (
          <div className="flex items-end gap-3 border rounded-md p-3">
            <div className="space-y-1.5 flex-1">
              <Label htmlFor="new-docker-name">Name</Label>
              <Input
                id="new-docker-name"
                value={newHost.name}
                onChange={(e) =>
                  setNewHost((prev) => ({ ...prev, name: e.target.value }))
                }
                placeholder="my-server"
                className="h-8"
              />
            </div>
            <div className="space-y-1.5 flex-[2]">
              <Label htmlFor="new-docker-host">Host</Label>
              <Input
                id="new-docker-host"
                value={newHost.host}
                onChange={(e) =>
                  setNewHost((prev) => ({ ...prev, host: e.target.value }))
                }
                placeholder="ssh://root@10.0.0.1"
                className="h-8"
              />
            </div>
            <div className="flex gap-1">
              <Button size="sm" onClick={handleAddHost}>
                Add
              </Button>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  setIsAdding(false);
                  setNewHost({ name: "", host: "" });
                }}
              >
                Cancel
              </Button>
            </div>
          </div>
        )}

        <div className="flex items-center gap-2">
          {!isAdding && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => setIsAdding(true)}
            >
              Add host
            </Button>
          )}
          {hasChanges && (
            <Button
              size="sm"
              disabled={updateMutation.isPending}
              onClick={handleSave}
            >
              {updateMutation.isPending ? (
                <>
                  <Spinner className="size-3" />
                  Saving...
                </>
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
