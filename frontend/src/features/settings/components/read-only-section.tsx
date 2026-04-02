import { toast } from "sonner";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";

import { useUpdateReadOnly } from "../hooks/use-settings";
import type { ReadOnlyConfig } from "../types";
import { EnvBadge } from "./env-badge";

interface ReadOnlySectionProps {
  config: ReadOnlyConfig;
}

export function ReadOnlySection({ config }: ReadOnlySectionProps) {
  const isEnv = config.source === "env";
  const mutation = useUpdateReadOnly();

  function handleToggle(checked: boolean) {
    mutation.mutate(checked, {
      onSuccess: (msg) => toast.success(msg),
      onError: (err) => toast.error(err.message),
    });
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-3">
          <CardTitle>Read-Only Mode</CardTitle>
          {isEnv && <EnvBadge />}
        </div>
        <CardDescription>
          When enabled, container actions (start, stop, restart, remove) are
          disabled. Log viewing is unaffected.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-3">
          <Switch
            id="read-only"
            checked={config.value}
            onCheckedChange={handleToggle}
            disabled={isEnv || mutation.isPending}
          />
          <Label htmlFor="read-only" className="cursor-pointer">
            {config.value ? "Enabled" : "Disabled"}
          </Label>
        </div>
      </CardContent>
    </Card>
  );
}
