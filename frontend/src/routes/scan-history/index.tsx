import { createFileRoute } from "@tanstack/react-router";

import { ScanHistoryPage } from "@/features/scanner/components/scan-history-page";
import { requireAuthIfEnabled } from "@/lib/auth-guard";

export const Route = createFileRoute("/scan-history/")({
  beforeLoad: async () => {
    await requireAuthIfEnabled();
  },
  component: ScanHistoryRoute,
});

function ScanHistoryRoute() {
  return (
    <main className="container mx-auto px-4 py-8">
      <ScanHistoryPage />
    </main>
  );
}
