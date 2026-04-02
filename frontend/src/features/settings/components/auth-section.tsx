import { useState } from "react";
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
import { Switch } from "@/components/ui/switch";

import { useUpdateAuth } from "../hooks/use-settings";
import type { AuthConfig } from "../types";
import { EnvBadge } from "./env-badge";

interface AuthSectionProps {
  config: AuthConfig;
}

export function AuthSection({ config }: AuthSectionProps) {
  const isEnv = config.source === "env";
  const [enabled, setEnabled] = useState(config.enabled);
  const [username, setUsername] = useState(config.adminUsername ?? "");
  const [password, setPassword] = useState("");

  const mutation = useUpdateAuth();

  const hasChanges =
    enabled !== config.enabled ||
    username !== (config.adminUsername ?? "") ||
    password.length > 0;

  function handleSave() {
    const trimmedUsername = username.trim();

    if (enabled && !trimmedUsername) {
      toast.error("Username is required when enabling auth");
      return;
    }
    if (enabled && !config.passwordConfigured && !password) {
      toast.error("Password is required when first enabling auth");
      return;
    }

    mutation.mutate(
      {
        enabled,
        adminUsername: trimmedUsername,
        ...(password ? { newPassword: password } : {}),
      },
      {
        onSuccess: (msg) => {
          toast.success(msg);
          setUsername(trimmedUsername);
          setPassword("");
        },
        onError: (err) => toast.error(err.message),
      },
    );
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-3">
          <CardTitle>Authentication</CardTitle>
          {isEnv && <EnvBadge />}
        </div>
        <CardDescription>
          Protect the dashboard with username and password authentication.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center gap-3">
          <Switch
            id="auth-enabled"
            checked={enabled}
            onCheckedChange={(checked) => {
              setEnabled(checked);
              if (!checked) setPassword("");
            }}
            disabled={isEnv}
          />
          <Label htmlFor="auth-enabled" className="cursor-pointer">
            {enabled ? "Enabled" : "Disabled"}
          </Label>
        </div>

        {enabled && (
          <div className="space-y-3 max-w-sm">
            <div className="space-y-1.5">
              <Label htmlFor="admin-username">Admin username</Label>
              <Input
                id="admin-username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                disabled={isEnv}
                placeholder="admin"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="admin-password">
                {config.enabled ? "New password (leave blank to keep current)" : "Password"}
              </Label>
              <Input
                id="admin-password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={isEnv}
                placeholder={config.enabled ? "Leave blank to keep current" : "Enter password"}
              />
            </div>
          </div>
        )}

        {hasChanges && !isEnv && (
          <Button
            size="sm"
            disabled={mutation.isPending}
            onClick={handleSave}
          >
            {mutation.isPending ? (
              <>
                <Spinner className="size-3" />
                Saving...
              </>
            ) : (
              "Save changes"
            )}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
