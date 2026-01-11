import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Edit2, Plus, Save, Trash2, Upload, X } from "lucide-react";
import { useId, useRef, useState } from "react";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import { getContainerEnvVariables } from "../api/get-container-env-variables";
import { updateContainerEnvVariables } from "../api/update-container-env-variables";

interface EnvironmentVariablesProps {
  containerId: string;
  containerHost: string;
  isReadOnly?: boolean;
  onContainerIdChange?: (newContainerId: string) => void;
}

// Parse .env file content into key-value pairs
function parseEnvFile(content: string): Record<string, string> {
  const env: Record<string, string> = {};
  const lines = content.split("\n");

  for (const line of lines) {
    const trimmed = line.trim();

    // Skip empty lines and comments
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    // Find the first = sign
    const equalIndex = trimmed.indexOf("=");
    if (equalIndex === -1) {
      continue;
    }

    const key = trimmed.substring(0, equalIndex).trim();
    let value = trimmed.substring(equalIndex + 1).trim();

    // Remove quotes if present
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1);
    }

    if (key) {
      env[key] = value;
    }
  }

  return env;
}

export function EnvironmentVariables({
  containerId,
  containerHost,
  isReadOnly = false,
  onContainerIdChange,
}: EnvironmentVariablesProps) {
  const queryClient = useQueryClient();
  const [isEditing, setIsEditing] = useState(false);
  const [editedEnv, setEditedEnv] = useState<Record<string, string>>({});
  const [deletedKeys, setDeletedKeys] = useState<Set<string>>(new Set());
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [showAddNew, setShowAddNew] = useState(false);
  const [showConfirmDialog, setShowConfirmDialog] = useState(false);
  const [modifiedKeys, setModifiedKeys] = useState<Set<string>>(new Set());
  const [showUploadPreview, setShowUploadPreview] = useState(false);
  const [parsedEnvFile, setParsedEnvFile] = useState<Record<string, string>>(
    {}
  );
  const fileInputRef = useRef<HTMLInputElement>(null);
  const newKeyId = useId();
  const newValueId = useId();

  const {
    data: envVariables,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["container-env", containerId, containerHost],
    queryFn: () => getContainerEnvVariables(containerId, containerHost),
    enabled: !!containerId && !!containerHost,
  });

  const updateMutation = useMutation({
    mutationFn: (env: Record<string, string>) =>
      updateContainerEnvVariables(containerId, containerHost, env),
    onSuccess: (newContainerId) => {
      // Invalidate queries for BOTH old and new container IDs
      queryClient.invalidateQueries({
        queryKey: ["container-env", containerId],
      });
      queryClient.invalidateQueries({
        queryKey: ["container-env", newContainerId],
      });
      queryClient.invalidateQueries({
        queryKey: ["containers"],
      });

      // Notify parent of new container ID
      onContainerIdChange?.(newContainerId);

      setIsEditing(false);
      setEditedEnv({});
      setDeletedKeys(new Set());
      setModifiedKeys(new Set());
      toast.success("Environment variables updated successfully", {
        description:
          "The container has been recreated with the new environment variables.",
      });
    },
    onError: (error: Error) => {
      toast.error("Failed to update environment variables", {
        description: error.message,
      });
    },
  });

  const handleEdit = () => {
    setEditedEnv({ ...envVariables });
    setDeletedKeys(new Set());
    setModifiedKeys(new Set());
    setIsEditing(true);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setEditedEnv({});
    setDeletedKeys(new Set());
    setModifiedKeys(new Set());
    setShowAddNew(false);
    setNewKey("");
    setNewValue("");
  };

  const handleSave = () => {
    setShowConfirmDialog(true);
  };

  const handleConfirmUpdate = () => {
    const finalEnv = { ...editedEnv };
    // Remove deleted keys
    deletedKeys.forEach((key) => {
      delete finalEnv[key];
    });
    updateMutation.mutate(finalEnv);
    setShowConfirmDialog(false);
  };

  const handleDelete = (key: string) => {
    setDeletedKeys((prev) => new Set(prev).add(key));
  };

  const handleValueChange = (key: string, value: string) => {
    setEditedEnv((prev) => ({ ...prev, [key]: value }));
  };

  const handleAddNew = () => {
    if (newKey.trim() && !editedEnv[newKey]) {
      setEditedEnv((prev) => ({ ...prev, [newKey]: newValue }));
      setModifiedKeys((prev) => new Set(prev).add(newKey));
      setNewKey("");
      setNewValue("");
      setShowAddNew(false);
    } else if (editedEnv[newKey]) {
      toast.error("Key already exists");
    } else {
      toast.error("Key cannot be empty");
    }
  };

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      try {
        const parsed = parseEnvFile(content);
        setParsedEnvFile(parsed);
        setShowUploadPreview(true);
      } catch {
        toast.error("Failed to parse .env file");
      }
    };
    reader.readAsText(file);

    // Reset the input so the same file can be selected again
    event.target.value = "";
  };

  const handleConfirmUpload = () => {
    const newKeys = new Set<string>();

    // Merge parsed env file into editedEnv
    Object.entries(parsedEnvFile).forEach(([key, value]) => {
      setEditedEnv((prev) => ({ ...prev, [key]: value }));
      newKeys.add(key);
    });

    // Track all uploaded keys as modified
    setModifiedKeys((prev) => new Set([...prev, ...newKeys]));

    // Close dialog and reset
    setShowUploadPreview(false);
    setParsedEnvFile({});

    toast.success(
      `Imported ${Object.keys(parsedEnvFile).length} variables from .env file`
    );
  };

  const handleCancelUpload = () => {
    setShowUploadPreview(false);
    setParsedEnvFile({});
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-4 text-sm text-muted-foreground">
        Loading environment variables...
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-4 text-sm text-destructive">
        Failed to load environment variables
      </div>
    );
  }

  const displayEnv = isEditing ? editedEnv : envVariables || {};
  const envEntries = Object.entries(displayEnv)
    .filter(([key]) => !deletedKeys.has(key))
    .sort(([keyA], [keyB]) => {
      // When editing, sort by modified status (modified first)
      if (isEditing) {
        const aIsModified = modifiedKeys.has(keyA);
        const bIsModified = modifiedKeys.has(keyB);

        if (aIsModified && !bIsModified) return -1;
        if (!aIsModified && bIsModified) return 1;
      }

      // For items with same modified status (or when not editing), maintain original order
      return 0;
    });

  if (envEntries.length === 0 && !isEditing) {
    return (
      <div className="space-y-3">
        <div className="flex items-center justify-center py-4 text-sm text-muted-foreground">
          No environment variables configured
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-3 pt-2">
      {/* Hidden file input */}
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileUpload}
        accept=".env"
        className="hidden"
      />

      {/* Edit/Save/Cancel buttons */}
      <div className="flex items-center gap-2 justify-end border-b pb-2">
        {!isEditing ? (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleEdit}
                  disabled={isReadOnly}
                  className="h-8"
                >
                  <Edit2 className="mr-2 h-3.5 w-3.5" />
                  Edit
                </Button>
              </TooltipTrigger>
              {isReadOnly && (
                <TooltipContent>
                  Cannot edit in read-only mode
                </TooltipContent>
              )}
            </Tooltip>
          </TooltipProvider>
        ) : (
          <>
            <Button
              variant="outline"
              size="sm"
              onClick={() => fileInputRef.current?.click()}
              className="h-8"
            >
              <Upload className="mr-2 h-3.5 w-3.5" />
              Upload .env
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowAddNew(true)}
              className="h-8"
            >
              <Plus className="mr-2 h-3.5 w-3.5" />
              Add
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleCancel}
              className="h-8"
            >
              <X className="mr-2 h-3.5 w-3.5" />
              Cancel
            </Button>
            <Button
              variant="default"
              size="sm"
              onClick={handleSave}
              disabled={updateMutation.isPending}
              className="h-8"
            >
              <Save className="mr-2 h-3.5 w-3.5" />
              {updateMutation.isPending ? "Saving..." : "Save"}
            </Button>
          </>
        )}
      </div>

      {/* Add new variable form */}
      {showAddNew && (
        <div className="rounded-lg border border-primary p-3 transition-colors bg-muted/30">
          <div className="flex items-end gap-3">
            <div className="flex-1 space-y-2">
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div className="space-y-1.5">
                  <Label htmlFor={newKeyId} className="text-xs font-medium">
                    Key
                  </Label>
                  <Input
                    id={newKeyId}
                    value={newKey}
                    onChange={(e) => setNewKey(e.target.value)}
                    placeholder="VARIABLE_NAME"
                    className="font-mono text-xs h-8"
                  />
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor={newValueId} className="text-xs font-medium">
                    Value
                  </Label>
                  <Input
                    id={newValueId}
                    value={newValue}
                    onChange={(e) => setNewValue(e.target.value)}
                    placeholder="value"
                    className="font-mono text-xs h-8"
                  />
                </div>
              </div>
            </div>
            <div className="flex gap-1">
              <Button
                variant="ghost"
                size="icon"
                onClick={handleAddNew}
                className="mb-0.5 h-8 w-8 text-primary hover:text-primary"
                title="Add variable"
              >
                <Plus className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => {
                  setShowAddNew(false);
                  setNewKey("");
                  setNewValue("");
                }}
                className="mb-0.5 h-8 w-8 text-muted-foreground"
                title="Cancel"
              >
                <X className="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Existing environment variables */}
      {envEntries.map(([key, value]) => {
        const isModified = modifiedKeys.has(key);
        return (
          <div
            key={key}
            className={`flex items-end gap-3 rounded-lg border p-3 transition-colors ${
              deletedKeys.has(key)
                ? "opacity-50 bg-destructive/10"
                : isModified && isEditing
                  ? "border-primary/50 bg-primary/5 hover:bg-primary/10"
                  : "hover:bg-muted/50"
            }`}
          >
            <div className="flex-1 space-y-2">
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div className="space-y-1.5">
                  <div className="flex items-center gap-2">
                    <Label
                      htmlFor={`key-${key}`}
                      className="text-xs font-medium"
                    >
                      Key
                    </Label>
                    {isModified && isEditing && (
                      <Badge
                        variant="default"
                        className="h-4 text-[10px] px-1.5"
                      >
                        New
                      </Badge>
                    )}
                  </div>
                  <Input
                    id={`key-${key}`}
                    value={key}
                    disabled
                    className="font-mono text-xs h-8"
                  />
                </div>
                <div className="space-y-1.5">
                  <Label
                    htmlFor={`value-${key}`}
                    className="text-xs font-medium"
                  >
                    Value
                  </Label>
                  <Input
                    id={`value-${key}`}
                    value={value}
                    onChange={(e) => handleValueChange(key, e.target.value)}
                    disabled={!isEditing}
                    className="font-mono text-xs h-8"
                  />
                </div>
              </div>
            </div>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => handleDelete(key)}
              className="mb-0.5 h-8 w-8 text-muted-foreground hover:text-destructive"
              disabled={!isEditing}
              title={
                isEditing ? "Mark for deletion" : "Edit to enable deletion"
              }
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        );
      })}

      {envEntries.length === 0 && isEditing && (
        <div className="flex items-center justify-center py-4 text-sm text-muted-foreground">
          No environment variables. Click "Add" to create one.
        </div>
      )}

      {/* Upload Preview Dialog */}
      <AlertDialog open={showUploadPreview} onOpenChange={setShowUploadPreview}>
        <AlertDialogContent className="max-w-3xl max-h-[80vh] overflow-hidden flex flex-col">
          <AlertDialogHeader className="shrink-0">
            <AlertDialogTitle>Preview .env File Import</AlertDialogTitle>
            <AlertDialogDescription>
              Review the variables that will be imported from the .env file.
            </AlertDialogDescription>
          </AlertDialogHeader>

          <div className="space-y-4 py-4 overflow-y-auto min-h-0">
            {/* Summary */}
            <div className="flex gap-4 text-sm">
              <div className="flex items-center gap-2">
                <span className="font-medium">Total:</span>
                <span className="px-2 py-0.5 rounded bg-muted">
                  {Object.keys(parsedEnvFile).length}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="font-medium">New:</span>
                <span className="px-2 py-0.5 rounded bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400">
                  {
                    Object.keys(parsedEnvFile).filter((key) => !editedEnv[key])
                      .length
                  }
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="font-medium">Updated:</span>
                <span className="px-2 py-0.5 rounded bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400">
                  {
                    Object.keys(parsedEnvFile).filter(
                      (key) =>
                        editedEnv[key] && editedEnv[key] !== parsedEnvFile[key]
                    ).length
                  }
                </span>
              </div>
            </div>

            {/* New Variables */}
            {Object.keys(parsedEnvFile).filter((key) => !editedEnv[key])
              .length > 0 && (
              <div className="space-y-2">
                <h4 className="text-sm font-semibold text-green-700 dark:text-green-400">
                  New Variables
                </h4>
                <div className="space-y-2 max-h-48 overflow-y-auto">
                  {Object.entries(parsedEnvFile)
                    .filter(([key]) => !editedEnv[key])
                    .map(([key, value]) => (
                      <div
                        key={key}
                        className="p-2 rounded border border-green-200 dark:border-green-900/30 bg-green-50 dark:bg-green-900/10"
                      >
                        <div className="font-mono text-xs break-all overflow-hidden">
                          <span className="font-semibold break-words">
                            {key}
                          </span>
                          <span className="text-muted-foreground"> = </span>
                          <span className="break-words">{value}</span>
                        </div>
                      </div>
                    ))}
                </div>
              </div>
            )}

            {/* Updated Variables */}
            {Object.keys(parsedEnvFile).filter(
              (key) => editedEnv[key] && editedEnv[key] !== parsedEnvFile[key]
            ).length > 0 && (
              <div className="space-y-2">
                <h4 className="text-sm font-semibold text-blue-700 dark:text-blue-400">
                  Updated Variables
                </h4>
                <div className="space-y-3 max-h-48 overflow-y-auto">
                  {Object.entries(parsedEnvFile)
                    .filter(
                      ([key]) =>
                        editedEnv[key] && editedEnv[key] !== parsedEnvFile[key]
                    )
                    .map(([key, value]) => (
                      <div
                        key={key}
                        className="p-2 rounded border border-blue-200 dark:border-blue-900/30 bg-blue-50 dark:bg-blue-900/10"
                      >
                        <div className="font-mono text-xs space-y-1 overflow-hidden">
                          <div className="font-semibold break-words">{key}</div>
                          <div className="flex flex-col gap-1">
                            <div className="flex items-start gap-2">
                              <span className="text-muted-foreground shrink-0">
                                Old:
                              </span>
                              <span className="text-red-600 dark:text-red-400 line-through break-all">
                                {editedEnv[key]}
                              </span>
                            </div>
                            <div className="flex items-start gap-2">
                              <span className="text-muted-foreground shrink-0">
                                New:
                              </span>
                              <span className="text-green-600 dark:text-green-400 break-all">
                                {value}
                              </span>
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                </div>
              </div>
            )}

            {/* Unchanged Variables */}
            {Object.keys(parsedEnvFile).filter(
              (key) => editedEnv[key] && editedEnv[key] === parsedEnvFile[key]
            ).length > 0 && (
              <div className="space-y-2">
                <h4 className="text-sm font-semibold text-muted-foreground">
                  Unchanged Variables (
                  {
                    Object.keys(parsedEnvFile).filter(
                      (key) =>
                        editedEnv[key] && editedEnv[key] === parsedEnvFile[key]
                    ).length
                  }
                  )
                </h4>
              </div>
            )}
          </div>

          <AlertDialogFooter className="shrink-0">
            <AlertDialogCancel onClick={handleCancelUpload}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmUpload}>
              Import Variables
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Confirmation Dialog */}
      <AlertDialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Update environment variables?</AlertDialogTitle>
            <AlertDialogDescription>
              Changing environment variables requires recreating the container.
              This will cause a brief downtime. Are you sure you want to
              continue?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmUpdate}>
              Confirm & Update
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
