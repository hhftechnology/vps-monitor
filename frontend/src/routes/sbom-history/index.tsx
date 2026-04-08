import { createFileRoute } from "@tanstack/react-router";

import { SBOMHistoryPage } from "@/features/scanner/components/sbom-history-page";
import { requireAuthIfEnabled } from "@/lib/auth-guard";

export const Route = createFileRoute("/sbom-history/")({
  beforeLoad: async () => {
    await requireAuthIfEnabled();
  },
  component: SBOMHistoryRoute,
});

function SBOMHistoryRoute() {
  return (
    <main className="container mx-auto px-4 py-8">
      <SBOMHistoryPage />
    </main>
  );
}
