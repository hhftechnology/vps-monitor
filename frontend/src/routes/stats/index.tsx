import { createFileRoute } from "@tanstack/react-router";

import { StatsPage } from "@/features/stats/components/stats-page";
import { requireAuthIfEnabled } from "@/lib/auth-guard";

export const Route = createFileRoute("/stats/")({
  beforeLoad: async () => {
    await requireAuthIfEnabled();
  },
  component: Stats,
});

function Stats() {
  return (
    <main className="container mx-auto px-4 py-8">
      <StatsPage />
    </main>
  );
}
