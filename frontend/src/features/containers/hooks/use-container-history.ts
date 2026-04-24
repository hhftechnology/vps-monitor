import { useQuery } from "@tanstack/react-query";

import { getContainerHistory } from "../api/get-container-history";

export function useContainerHistory(id: string, host: string, enabled = true) {
	return useQuery({
		queryKey: ["container-history", host, id],
		queryFn: () => getContainerHistory(id, host),
		enabled: enabled && Boolean(id) && Boolean(host),
		staleTime: 60_000,
		refetchInterval: 60_000,
	});
}
