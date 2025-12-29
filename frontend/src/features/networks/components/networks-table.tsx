import { useMemo, useState } from "react";
import {
  GlobeIcon,
  InfoIcon,
  LockIcon,
  RefreshCcwIcon,
  SearchIcon,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import { useNetworksQuery } from "../hooks/use-networks-query";
import { NetworkDetailsSheet } from "./network-details-sheet";
import type { NetworkInfo } from "../types";

export function NetworksTable() {
  const { data, isLoading, error, refetch, isRefetching } = useNetworksQuery();

  const [searchText, setSearchText] = useState("");
  const [selectedNetwork, setSelectedNetwork] = useState<{
    network: NetworkInfo;
    host: string;
  } | null>(null);

  // Flatten networks from all hosts
  const allNetworks = useMemo(() => {
    if (!data?.networks) return [];
    const networks: Array<NetworkInfo & { hostName: string }> = [];
    for (const [hostName, hostNetworks] of Object.entries(data.networks)) {
      for (const net of hostNetworks) {
        networks.push({ ...net, hostName });
      }
    }
    return networks;
  }, [data?.networks]);

  // Filter networks by search
  const filteredNetworks = useMemo(() => {
    if (!searchText) return allNetworks;
    const search = searchText.toLowerCase();
    return allNetworks.filter((net) => {
      const name = net.name.toLowerCase();
      const driver = net.driver.toLowerCase();
      return name.includes(search) || driver.includes(search);
    });
  }, [allNetworks, searchText]);

  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-8">
          <Spinner className="mr-2 size-4" />
          Loading networks...
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-destructive">
          Failed to load networks: {error.message}
        </CardContent>
      </Card>
    );
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Docker Networks</CardTitle>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => refetch()}
                  disabled={isRefetching}
                >
                  <RefreshCcwIcon
                    className={`size-4 ${isRefetching ? "animate-spin" : ""}`}
                  />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Refresh</TooltipContent>
            </Tooltip>
          </div>
          <div className="relative mt-4">
            <SearchIcon className="absolute left-2 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
            <Input
              placeholder="Search networks..."
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              className="pl-8"
            />
          </div>
        </CardHeader>
        <CardContent>
          {filteredNetworks.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              {searchText
                ? "No networks match your search"
                : "No networks found"}
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Host</TableHead>
                  <TableHead>Driver</TableHead>
                  <TableHead>Scope</TableHead>
                  <TableHead>Containers</TableHead>
                  <TableHead className="w-[80px]">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredNetworks.map((network) => (
                  <TableRow key={`${network.hostName}-${network.id}`}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{network.name}</span>
                        {network.internal ? (
                          <Tooltip>
                            <TooltipTrigger>
                              <LockIcon className="size-3 text-muted-foreground" />
                            </TooltipTrigger>
                            <TooltipContent>Internal network</TooltipContent>
                          </Tooltip>
                        ) : (
                          <Tooltip>
                            <TooltipTrigger>
                              <GlobeIcon className="size-3 text-muted-foreground" />
                            </TooltipTrigger>
                            <TooltipContent>External network</TooltipContent>
                          </Tooltip>
                        )}
                        {network.enable_ipv6 && (
                          <Badge variant="outline" className="text-xs">
                            IPv6
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{network.hostName}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary">{network.driver}</Badge>
                    </TableCell>
                    <TableCell>{network.scope}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{network.containers}</Badge>
                    </TableCell>
                    <TableCell>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() =>
                              setSelectedNetwork({
                                network,
                                host: network.hostName,
                              })
                            }
                          >
                            <InfoIcon className="size-4" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>View details</TooltipContent>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <NetworkDetailsSheet
        network={selectedNetwork?.network ?? null}
        host={selectedNetwork?.host ?? ""}
        isOpen={!!selectedNetwork}
        onOpenChange={(open) => !open && setSelectedNetwork(null)}
      />
    </>
  );
}
