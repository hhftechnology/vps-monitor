import { useQuery } from "@tanstack/react-query";

import { getSystemStats } from "../api/get-system-stats";

export function useSystemStats() {
  return useQuery({
    queryKey: ["system-stats"],
    queryFn: getSystemStats,
    refetchInterval: 2000, // Refresh every 2 seconds
  });
}
