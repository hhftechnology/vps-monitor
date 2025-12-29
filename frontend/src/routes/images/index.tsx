import { createFileRoute } from "@tanstack/react-router";

import { ImagesTable } from "@/features/images/components/images-table";
import { requireAuthIfEnabled } from "@/lib/auth-guard";

export const Route = createFileRoute("/images/")({
  beforeLoad: async () => {
    await requireAuthIfEnabled();
  },
  component: ImagesPage,
});

function ImagesPage() {
  return (
    <main className="container mx-auto px-4 py-8">
      <ImagesTable />
    </main>
  );
}
