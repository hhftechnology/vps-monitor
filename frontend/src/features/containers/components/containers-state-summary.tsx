import type { StateCounts } from "./container-utils";

interface ContainersStateSummaryProps {
  stateCounts: StateCounts;
}

export function ContainersStateSummary({
  stateCounts,
}: ContainersStateSummaryProps) {
  return (
    <div className="flex flex-wrap items-center gap-3 text-sm">
      {stateCounts.running > 0 && (
        <div className="flex items-center gap-2 rounded-md border bg-card px-3 py-2">
          <div className="size-2 rounded-full bg-emerald-500" />
          <span className="text-muted-foreground">Running</span>
          <span className="font-semibold">{stateCounts.running}</span>
        </div>
      )}
      {stateCounts.exited > 0 && (
        <div className="flex items-center gap-2 rounded-md border bg-card px-3 py-2">
          <div className="size-2 rounded-full bg-muted" />
          <span className="text-muted-foreground">Exited</span>
          <span className="font-semibold">{stateCounts.exited}</span>
        </div>
      )}
      {stateCounts.paused > 0 && (
        <div className="flex items-center gap-2 rounded-md border bg-card px-3 py-2">
          <div className="size-2 rounded-full bg-amber-500" />
          <span className="text-muted-foreground">Paused</span>
          <span className="font-semibold">{stateCounts.paused}</span>
        </div>
      )}
      {stateCounts.restarting > 0 && (
        <div className="flex items-center gap-2 rounded-md border bg-card px-3 py-2">
          <div className="size-2 rounded-full bg-blue-500" />
          <span className="text-muted-foreground">Restarting</span>
          <span className="font-semibold">{stateCounts.restarting}</span>
        </div>
      )}
      {stateCounts.dead > 0 && (
        <div className="flex items-center gap-2 rounded-md border bg-card px-3 py-2">
          <div className="size-2 rounded-full bg-rose-500" />
          <span className="text-muted-foreground">Dead</span>
          <span className="font-semibold">{stateCounts.dead}</span>
        </div>
      )}
    </div>
  );
}
