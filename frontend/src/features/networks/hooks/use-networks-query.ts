import { useQuery } from "@tanstack/react-query";

import { getNetworkDetails, getNetworks } from "../api/get-networks";

export function useNetworksQuery() {
  return useQuery({
    queryKey: ["networks"],
    queryFn: getNetworks,
    staleTime: 30_000,
  });
}

export function useNetworkDetailsQuery(networkId: string, host: string) {
  return useQuery({
    queryKey: ["networks", networkId, host],
    queryFn: () => getNetworkDetails(networkId, host),
    enabled: !!networkId && !!host,
  });
}
