import { useQuery } from "@tanstack/react-query";

import { getContainers } from "../api/get-containers";

export function useContainersQuery() {
  return useQuery({
    queryKey: ["containers"],
    queryFn: getContainers,
    staleTime: 30_000,
  });
}
