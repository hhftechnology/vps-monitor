import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  acknowledgeAlert,
  getAlertConfig,
  getAlerts,
} from "../api/get-alerts";

export function useAlertsQuery() {
  return useQuery({
    queryKey: ["alerts"],
    queryFn: getAlerts,
    staleTime: 10_000, // Refresh more frequently for alerts
    refetchInterval: 30_000, // Auto-refresh every 30 seconds
  });
}

export function useAlertConfigQuery() {
  return useQuery({
    queryKey: ["alerts", "config"],
    queryFn: getAlertConfig,
    staleTime: 60_000,
  });
}

export function useAcknowledgeAlertMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (alertId: string) => acknowledgeAlert(alertId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
    },
  });
}
