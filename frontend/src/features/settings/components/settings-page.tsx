import { Spinner } from "@/components/ui/spinner";

import { useSettings } from "../hooks/use-settings";
import { AuthSection } from "./auth-section";
import { CoolifyHostsSection } from "./coolify-hosts-section";
import { DockerHostsSection } from "./docker-hosts-section";
import { ReadOnlySection } from "./read-only-section";

export function SettingsPage() {
  const { data, isLoading, error } = useSettings();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Spinner className="size-6" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="container mx-auto max-w-3xl px-4 py-8">
        <p className="text-sm text-destructive">
          Failed to load settings: {error.message}
        </p>
      </div>
    );
  }

  if (!data) return null;

  const dockerHostsKey = `${data.dockerHosts.source}-${data.dockerHosts.hosts.length}`;
  const coolifyHostsKey = `${data.coolifyHosts.source}-${data.coolifyHosts.hosts.length}`;

  return (
    <div className="container mx-auto max-w-3xl px-4 py-8 space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Manage VPS Monitor configuration. Sections marked as set via environment
          variable can only be changed by updating the environment and
          restarting.
        </p>
      </div>
      <DockerHostsSection key={dockerHostsKey} config={data.dockerHosts} />
      <CoolifyHostsSection key={coolifyHostsKey} config={data.coolifyHosts} />
      <ReadOnlySection config={data.readOnly} />
      <AuthSection key={`${data.auth.enabled}-${data.auth.adminUsername}`} config={data.auth} />
    </div>
  );
}
