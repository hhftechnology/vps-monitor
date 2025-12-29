import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
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

import { useNetworkDetailsQuery } from "../hooks/use-networks-query";
import type { NetworkInfo } from "../types";

interface NetworkDetailsSheetProps {
  network: NetworkInfo | null;
  host: string;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
}

export function NetworkDetailsSheet({
  network,
  host,
  isOpen,
  onOpenChange,
}: NetworkDetailsSheetProps) {
  const { data: details, isLoading, error } = useNetworkDetailsQuery(
    network?.id ?? "",
    host
  );

  return (
    <Sheet open={isOpen} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-xl w-full overflow-y-auto p-6">
        <SheetHeader>
          <SheetTitle>Network Details</SheetTitle>
          <SheetDescription>{network?.name}</SheetDescription>
        </SheetHeader>

        {isLoading && (
          <div className="flex items-center justify-center py-8">
            <Spinner className="mr-2 size-4" />
            Loading details...
          </div>
        )}

        {error && (
          <div className="py-8 text-center text-destructive">
            Failed to load details: {error.message}
          </div>
        )}

        {details && (
          <div className="mt-6 space-y-6">
            <Card>
              <CardContent>
                <div className="grid gap-3 text-sm">
                  <div className="grid grid-cols-3 gap-4">
                    <span className="text-muted-foreground">Name</span>
                    <span className="col-span-2 font-medium">
                      {details.name}
                    </span>
                  </div>
                  <div className="grid grid-cols-3 gap-4">
                    <span className="text-muted-foreground">ID</span>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="col-span-2 font-mono text-xs truncate cursor-help">
                          {details.id}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent className="max-w-md">
                        {details.id}
                      </TooltipContent>
                    </Tooltip>
                  </div>
                  <div className="grid grid-cols-3 gap-4">
                    <span className="text-muted-foreground">Host</span>
                    <span className="col-span-2">
                      <Badge variant="outline">{details.host}</Badge>
                    </span>
                  </div>
                  <div className="grid grid-cols-3 gap-4">
                    <span className="text-muted-foreground">Driver</span>
                    <span className="col-span-2">
                      <Badge variant="secondary">{details.driver}</Badge>
                    </span>
                  </div>
                  <div className="grid grid-cols-3 gap-4">
                    <span className="text-muted-foreground">Scope</span>
                    <span className="col-span-2 font-medium">
                      {details.scope}
                    </span>
                  </div>
                  <div className="grid grid-cols-3 gap-4">
                    <span className="text-muted-foreground">Internal</span>
                    <span className="col-span-2">
                      <Badge variant={details.internal ? "default" : "outline"}>
                        {details.internal ? "Yes" : "No"}
                      </Badge>
                    </span>
                  </div>
                  <div className="grid grid-cols-3 gap-4">
                    <span className="text-muted-foreground">IPv6</span>
                    <span className="col-span-2">
                      <Badge
                        variant={details.enable_ipv6 ? "default" : "outline"}
                      >
                        {details.enable_ipv6 ? "Enabled" : "Disabled"}
                      </Badge>
                    </span>
                  </div>
                  {details.created && (
                    <div className="grid grid-cols-3 gap-4">
                      <span className="text-muted-foreground">Created</span>
                      <span className="col-span-2 font-medium">
                        {details.created}
                      </span>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>

            {details.ipam && details.ipam.config && details.ipam.config.length > 0 && (
              <div className="space-y-3">
                <h3 className="text-sm font-medium">IPAM Configuration</h3>
                <Card>
                  <CardContent className="p-0">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Subnet</TableHead>
                          <TableHead>Gateway</TableHead>
                          <TableHead>IP Range</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {details.ipam.config.map((pool, index) => (
                          <TableRow key={index}>
                            <TableCell className="font-mono text-xs">
                              {pool.subnet || "-"}
                            </TableCell>
                            <TableCell className="font-mono text-xs">
                              {pool.gateway || "-"}
                            </TableCell>
                            <TableCell className="font-mono text-xs">
                              {pool.ip_range || "-"}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>
              </div>
            )}

            {details.connected_containers && details.connected_containers.length > 0 && (
              <div className="space-y-3">
                <h3 className="text-sm font-medium">
                  Connected Containers ({details.connected_containers.length})
                </h3>
                <Card>
                  <CardContent className="p-0">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Name</TableHead>
                          <TableHead>IPv4</TableHead>
                          <TableHead>MAC</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {details.connected_containers.map((container) => (
                          <TableRow key={container.container_id}>
                            <TableCell>
                              <div className="flex flex-col">
                                <span className="font-medium">
                                  {container.container_name}
                                </span>
                                <span className="text-xs text-muted-foreground font-mono">
                                  {container.container_id.slice(0, 12)}
                                </span>
                              </div>
                            </TableCell>
                            <TableCell className="font-mono text-xs">
                              {container.ipv4_address || "-"}
                            </TableCell>
                            <TableCell className="font-mono text-xs">
                              {container.mac_address || "-"}
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>
              </div>
            )}

            {details.options && Object.keys(details.options).length > 0 && (
              <div className="space-y-3">
                <h3 className="text-sm font-medium">Options</h3>
                <Card>
                  <CardContent>
                    <div className="space-y-2">
                      {Object.entries(details.options).map(([key, value]) => (
                        <div
                          key={key}
                          className="rounded-md bg-muted/30 p-2 text-xs"
                        >
                          <div className="font-semibold text-foreground">
                            {key}
                          </div>
                          <div className="font-mono text-muted-foreground">
                            {value}
                          </div>
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              </div>
            )}

            {details.labels && Object.keys(details.labels).length > 0 && (
              <div className="space-y-3">
                <h3 className="text-sm font-medium">Labels</h3>
                <Card>
                  <CardContent>
                    <div className="space-y-2">
                      {Object.entries(details.labels).map(([key, value]) => (
                        <div
                          key={key}
                          className="rounded-md bg-muted/30 p-2 text-xs"
                        >
                          <div className="font-semibold text-foreground">
                            {key}
                          </div>
                          <div className="font-mono text-muted-foreground break-all">
                            {value}
                          </div>
                        </div>
                      ))}
                    </div>
                  </CardContent>
                </Card>
              </div>
            )}
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
