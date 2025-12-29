import { FileTextIcon, PlayIcon, RotateCwIcon, SquareIcon, Trash2Icon } from "lucide-react";
import { Fragment } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import {
  formatContainerName,
  formatCreatedDate,
  formatUptime,
  getStateBadgeClass,
  toTitleCase,
} from "./container-utils";

import type { ContainerInfo } from "../types";

import type {
  ContainerActionType,
  GroupByOption,
  GroupedContainers,
} from "./container-utils";

interface ContainersTableProps {
  isLoading: boolean;
  isError: boolean;
  error: unknown;
  groupBy: GroupByOption;
  filteredContainers: ContainerInfo[];
  groupedItems: GroupedContainers[] | null;
  pageItems: ContainerInfo[];
  pendingAction: { id: string; type: ContainerActionType } | null;
  isReadOnly: boolean;
  onStart: (container: ContainerInfo) => void;
  onStop: (container: ContainerInfo) => void;
  onRestart: (container: ContainerInfo) => void;
  onDelete: (container: ContainerInfo) => void;
  onViewLogs: (container: ContainerInfo) => void;
  onRetry: () => void;
}

export function ContainersTable({
  isLoading,
  isError,
  error,
  groupBy,
  filteredContainers,
  groupedItems,
  pageItems,
  pendingAction,
  isReadOnly,
  onStart,
  onStop,
  onRestart,
  onDelete,
  onViewLogs,
  onRetry,
}: ContainersTableProps) {
  const isContainerActionPending = (
    action: ContainerActionType,
    containerId: string
  ) =>
    pendingAction?.id === containerId && pendingAction.type === action;

  const isContainerBusy = (containerId: string) =>
    pendingAction?.id === containerId;

  const renderContainerRow = (container: ContainerInfo) => {
    const state = container.state.toLowerCase();
    const busy = isContainerBusy(container.id);
    const startPending = isContainerActionPending("start", container.id);
    const stopPending = isContainerActionPending("stop", container.id);
    const restartPending = isContainerActionPending("restart", container.id);
    const removePending = isContainerActionPending("remove", container.id);

    return (
      <TableRow key={container.id} className="hover:bg-muted/50">
        <TableCell className="h-16 px-4 font-medium">
          {formatContainerName(container.names)}
        </TableCell>
        <TableCell className="h-16 px-4 text-sm text-muted-foreground">
          {container.image}
        </TableCell>
        <TableCell className="h-16 px-4">
          <Badge
            className={`${getStateBadgeClass(container.state)} border-0`}
          >
            {toTitleCase(container.state)}
          </Badge>
        </TableCell>
        <TableCell className="h-16 px-4 text-sm text-muted-foreground">
          {formatUptime(container.created)}
        </TableCell>
        <TableCell className="h-16 px-4 text-sm text-muted-foreground">
          {formatCreatedDate(container.created)}
        </TableCell>
        <TableCell className="h-16 px-4 max-w-[300px] text-sm text-muted-foreground">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="block cursor-help truncate">
                  {container.command}
                </span>
              </TooltipTrigger>
              <TooltipContent className="max-w-md">
                {container.command}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </TableCell>
        <TableCell className="h-16 px-4">
          <TooltipProvider>
            <div className="flex items-center gap-1">
              {state === "exited" && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-block">
                      <Button
                        variant="outline"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => onStart(container)}
                        disabled={busy || isReadOnly}
                      >
                        {startPending ? (
                          <Spinner className="size-4" />
                        ) : (
                          <PlayIcon className="size-4" />
                        )}
                      </Button>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    {isReadOnly ? "Start (Read-only mode)" : "Start"}
                  </TooltipContent>
                </Tooltip>
              )}
              {state === "running" && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span className="inline-block">
                      <Button
                        variant="outline"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => onStop(container)}
                        disabled={busy || isReadOnly}
                      >
                        {stopPending ? (
                          <Spinner className="size-4" />
                        ) : (
                          <SquareIcon className="size-4" />
                        )}
                      </Button>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    {isReadOnly ? "Stop (Read-only mode)" : "Stop"}
                  </TooltipContent>
                </Tooltip>
              )}
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="inline-block">
                    <Button
                      variant="outline"
                      size="icon"
                      className="h-8 w-8"
                      onClick={() => onRestart(container)}
                      disabled={busy || isReadOnly}
                    >
                      {restartPending ? (
                        <Spinner className="size-4" />
                      ) : (
                        <RotateCwIcon className="size-4" />
                      )}
                    </Button>
                  </span>
                </TooltipTrigger>
                <TooltipContent>
                  {isReadOnly ? "Restart (Read-only mode)" : "Restart"}
                </TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span className="inline-block">
                    <Button
                      variant="outline"
                      size="icon"
                      className="h-8 w-8 text-destructive hover:bg-destructive hover:text-white"
                      onClick={() => onDelete(container)}
                      disabled={busy || isReadOnly}
                    >
                      {removePending ? (
                        <Spinner className="size-4" />
                      ) : (
                        <Trash2Icon className="size-4" />
                      )}
                    </Button>
                  </span>
                </TooltipTrigger>
                <TooltipContent>
                  {isReadOnly ? "Delete (Read-only mode)" : "Delete"}
                </TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => onViewLogs(container)}
                    disabled={busy}
                  >
                    <FileTextIcon className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>View Logs</TooltipContent>
              </Tooltip>
            </div>
          </TooltipProvider>
        </TableCell>
      </TableRow>
    );
  };

  return (
    <div className="rounded-lg border bg-card">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent border-b">
            <TableHead className="h-12 px-4 font-medium">Name</TableHead>
            <TableHead className="h-12 px-4 font-medium">Image</TableHead>
            <TableHead className="h-12 px-4 font-medium w-[120px]">
              State
            </TableHead>
            <TableHead className="h-12 px-4 font-medium">Uptime</TableHead>
            <TableHead className="h-12 px-4 font-medium">Created</TableHead>
            <TableHead className="h-12 px-4 font-medium">Command</TableHead>
            <TableHead className="h-12 px-4 font-medium w-[120px]">
              Actions
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {isLoading ? (
            <TableRow>
              <TableCell colSpan={7} className="h-32">
                <div className="flex items-center justify-center text-sm text-muted-foreground">
                  <Spinner className="mr-2" />
                  Loading containers…
                </div>
              </TableCell>
            </TableRow>
          ) : isError ? (
            <TableRow>
              <TableCell colSpan={7} className="h-32">
                <div className="flex flex-col items-center gap-3 text-center">
                  <p className="text-sm text-muted-foreground">
                    {(error as Error)?.message || "Unable to load containers."}
                  </p>
                  <Button size="sm" variant="outline" onClick={onRetry}>
                    Try again
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ) : filteredContainers.length === 0 ? (
            <TableRow>
              <TableCell colSpan={7} className="h-32">
                <div className="text-center text-sm text-muted-foreground">
                  No containers found.
                </div>
              </TableCell>
            </TableRow>
          ) : groupBy === "compose" && groupedItems ? (
            groupedItems.map((group) => (
              <Fragment key={group.project}>
                <TableRow className="bg-muted/30 hover:bg-muted/30">
                  <TableCell
                    colSpan={7}
                    className="h-10 px-4 text-xs font-medium text-muted-foreground"
                  >
                    {group.project} · {group.items.length} container
                    {group.items.length === 1 ? "" : "s"}
                  </TableCell>
                </TableRow>
                {group.items.map(renderContainerRow)}
              </Fragment>
            ))
          ) : (
            pageItems.map(renderContainerRow)
          )}
        </TableBody>
      </Table>
    </div>
  );
}
