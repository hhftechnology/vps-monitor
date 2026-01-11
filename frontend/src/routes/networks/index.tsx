import { createFileRoute } from "@tanstack/react-router";

import { NetworksTable } from "@/features/networks/components/networks-table";
import { requireAuthIfEnabled } from "@/lib/auth-guard";

export const Route = createFileRoute("/networks/")({
  beforeLoad: async () => {
    await requireAuthIfEnabled();
  },
  component: NetworksPage,
});

function NetworksPage() {
  return (
    <main className="container mx-auto px-4 py-8">
      <NetworksTable />
    </main>
  );
}
