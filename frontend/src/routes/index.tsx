import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { ContainersDashboard } from "@/features/containers/components/containers-dashboard";
import { requireAuthIfEnabled } from "@/lib/auth-guard";

// Define search params schema for the dashboard
const dashboardSearchSchema = z
  .object({
    search: z.string().optional(),
    state: z.string().optional(),
    sort: z.enum(["asc", "desc"]).optional(),
    group: z.enum(["none", "compose"]).optional(),
    page: z.number().optional(),
    pageSize: z.number().optional(),
    from: z.string().optional(),
    to: z.string().optional(),
  })
  .passthrough()
  .catch({});

export const Route = createFileRoute("/")({
  validateSearch: dashboardSearchSchema.parse,
  beforeLoad: async () => {
    await requireAuthIfEnabled();
  },
  component: Index,
});

function Index() {
  return (
    <main className="container mx-auto px-4 py-8">
      <ContainersDashboard />
    </main>
  );
}
