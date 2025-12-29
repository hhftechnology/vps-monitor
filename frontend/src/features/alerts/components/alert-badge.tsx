import { BellIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";

import { useAlertsQuery } from "../hooks/use-alerts-query";

export function AlertBadge() {
  const { data } = useAlertsQuery();

  const unacknowledgedCount = data?.alerts?.filter(
    (alert) => !alert.acknowledged
  ).length ?? 0;

  if (unacknowledgedCount === 0) {
    return <BellIcon className="size-4" />;
  }

  return (
    <span className="relative">
      <BellIcon className="size-4" />
      <Badge
        variant="destructive"
        className="absolute -top-2 -right-2 h-4 w-4 p-0 flex items-center justify-center text-[10px]"
      >
        {unacknowledgedCount > 9 ? "9+" : unacknowledgedCount}
      </Badge>
    </span>
  );
}
