import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
  ArrowDownIcon,
  ArrowDownToLineIcon,
  ArrowLeftIcon,
  CheckIcon,
  ChevronDownIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  CopyIcon,
  DownloadIcon,
  EyeIcon,
  EyeOffIcon,
  FilterIcon,
  PlayIcon,
  RefreshCcwIcon,
  SearchIcon,
  SquareIcon,
  TerminalIcon,
  WrapTextIcon
} from "lucide-react";
import {
  useCallback,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState
} from "react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger
} from "@/components/ui/popover";
import { Spinner } from "@/components/ui/spinner";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger
} from "@/components/ui/tooltip";
import {
  getContainerLogsParsed,
  getLogLevelBadgeColor,
  streamContainerLogsParsed
} from "@/features/containers/api/get-container-logs-parsed";
import { getContainers } from "@/features/containers/api/get-containers";
import {
  formatContainerName,
  formatCreatedDate,
  formatUptime,
  getStateBadgeClass,
  toTitleCase
} from "@/features/containers/components/container-utils";
import { EnvironmentVariables } from "@/features/containers/components/environment-variables";
import { Terminal } from "@/features/containers/components/terminal";
import { requireAuthIfEnabled } from "@/lib/auth-guard";

import type {
  LogEntry,
  LogLevel,
} from "@/features/containers/api/get-container-logs-parsed";
export const Route = createFileRoute("/containers/$containerId/logs")({
  beforeLoad: async () => {
    await requireAuthIfEnabled();
  },
  component: ContainerLogsPage,
});

function ContainerLogsPage() {
  const { containerId: encodedContainerId } = Route.useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [logLines, setLogLines] = useState(100);
  const [isStreaming, setIsStreaming] = useState(false);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [isLoadingLogs, setIsLoadingLogs] = useState(false);
  const [searchText, setSearchText] = useState("");
  const [selectedLevels, setSelectedLevels] = useState<Set<LogLevel>>(
    new Set()
  );
  const [showTimestamps, setShowTimestamps] = useState(true);
  const [autoScroll, setAutoScroll] = useState(true);
  const [wrapText, setWrapText] = useState(false);
  const [showFilters, setShowFilters] = useState(false);
  const [showLabels, setShowLabels] = useState(false);
  const [showEnvVariables, setShowEnvVariables] = useState(false);
  const [showTerminal, setShowTerminal] = useState(false);
  const [currentMatchIndex, setCurrentMatchIndex] = useState(0);
  const abortControllerRef = useRef<AbortController | null>(null);
  const parentRef = useRef<HTMLDivElement>(null);
  const autoScrollRef = useRef(autoScroll);
  const logLinesInputId = useId();

  // Keep ref in sync with state
  useEffect(() => {
    autoScrollRef.current = autoScroll;
  }, [autoScroll]);

  // Decode the URL parameter (could be name or ID)
  const containerIdentifier = decodeURIComponent(encodedContainerId);

  // Fetch container info
  const { data: containersData } = useQuery({
    queryKey: ["containers"],
    queryFn: getContainers,
  });

  const containers = containersData?.containers ?? [];

  // Find container by name (preferred) or ID (fallback for backward compatibility)
  const container = containers.find((c) => {
    // Check if identifier matches the container name (without leading slash)
    if (c.names && c.names.length > 0) {
      const cleanName = c.names[0].startsWith("/")
        ? c.names[0].slice(1)
        : c.names[0];
      if (cleanName === containerIdentifier) {
        return true;
      }
    }
    // Fallback: check if it matches the ID (full or short)
    return c.id === containerIdentifier || c.id.startsWith(containerIdentifier);
  });

  // Use the actual container ID for API calls (Docker API accepts both name and ID, but we'll use ID for consistency)
  const actualContainerId = container?.id || containerIdentifier;

  const handleContainerRecreated = async (_newContainerId: string) => {
    await queryClient.invalidateQueries({ queryKey: ["containers"] });
    if (isStreaming) {
      stopStreaming();
      await new Promise((resolve) => setTimeout(resolve, 100));
      void startStreaming();
    } else {
      // If not streaming, just refetch logs
      await fetchLogs();
    }
  };

  const scrollToBottom = useCallback(() => {
    if (autoScrollRef.current && parentRef.current) {
      // For virtualized list, scroll the parent container to bottom
      parentRef.current.scrollTop = parentRef.current.scrollHeight;
    }
  }, []);

  const fetchLogs = useCallback(async () => {
    if (!actualContainerId || !container?.host) return;

    setIsLoadingLogs(true);
    try {
      const logEntries = await getContainerLogsParsed(actualContainerId, container.host, {
        tail: logLines,
      });
      setLogs(logEntries);
      setTimeout(scrollToBottom, 100);
    } catch (error) {
      if (error instanceof Error) {
        toast.error(`Failed to fetch logs: ${error.message}`);
      }
      setLogs([]);
    } finally {
      setIsLoadingLogs(false);
    }
  }, [actualContainerId, container?.host, logLines, scrollToBottom]);

  const startStreaming = useCallback(async () => {
    if (!actualContainerId || !container?.host) return;

    setIsStreaming(true);
    setIsLoadingLogs(true);
    setLogs([]);

    try {
      const abortController = new AbortController();
      abortControllerRef.current = abortController;

      const stream = streamContainerLogsParsed(
        actualContainerId,
        container.host,
        {
          tail: logLines,
        },
        abortController.signal
      );

      setIsLoadingLogs(false);

      for await (const entry of stream) {
        if (abortController.signal.aborted) {
          break;
        }

        setLogs((prev) => [...prev, entry]);
        setTimeout(scrollToBottom, 100);
      }
    } catch (error) {
      if (error instanceof Error) {
        const message = error.message.toLowerCase();
        const isAbort =
          error.name === "AbortError" || message.includes("aborted");
        if (!isAbort) {
          toast.error(`Failed to start streaming: ${error.message}`);
        }
      }
      setIsStreaming(false);
    } finally {
      setIsLoadingLogs(false);
      abortControllerRef.current = null;
    }
  }, [actualContainerId, container?.host, logLines, scrollToBottom]);

  const stopStreaming = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
    }
    setIsStreaming(false);
  }, []);

  const handleToggleStream = () => {
    if (isStreaming) {
      stopStreaming();
    } else {
      startStreaming();
    }
  };

  const handleRefresh = () => {
    if (!isStreaming) {
      fetchLogs();
    }
  };

  useEffect(() => {
    fetchLogs();
    return () => {
      stopStreaming();
    };
  }, [fetchLogs, stopStreaming]);

  useEffect(() => {
    if (!isStreaming) {
      fetchLogs();
    }
  }, [isStreaming, fetchLogs]);

  const handleLogLinesChange = (value: string) => {
    const num = parseInt(value, 10);
    if (!Number.isNaN(num) && num > 0) {
      setLogLines(num);
    }
  };

  const toggleLogLevel = (level: LogLevel) => {
    setSelectedLevels((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(level)) {
        newSet.delete(level);
      } else {
        newSet.add(level);
      }
      return newSet;
    });
  };

  const handleCopyLog = (entry: LogEntry) => {
    const text = entry.message || entry.raw || "";
    navigator.clipboard.writeText(text);
    toast.success("Log entry copied to clipboard");
  };

  const handleDownloadLogs = (format: "json" | "txt") => {
    if (filteredLogs.length === 0) {
      toast.error("No logs to download");
      return;
    }

    const filename = `${container?.names?.[0] || "container"}-logs-${new Date().toISOString()}.${format}`;
    let content: string;
    let mimeType: string;

    if (format === "json") {
      content = JSON.stringify(filteredLogs, null, 2);
      mimeType = "application/json";
    } else {
      content = filteredLogs
        .map((entry) => {
          const timestamp = entry.timestamp
            ? new Date(entry.timestamp).toISOString()
            : "";
          const level = entry.level || "UNKNOWN";
          const message = entry.message || entry.raw || "";
          return `[${timestamp}] [${level}] ${message}`;
        })
        .join("\n");
      mimeType = "text/plain";
    }

    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    toast.success(`Logs downloaded as ${format.toUpperCase()}`);
  };

  // Filter logs by level only (search no longer filters, just highlights)
  const filteredLogs = useMemo(() => {
    return logs.filter((entry) => {
      // Filter by log level only
      if (selectedLevels.size > 0 && entry.level) {
        if (!selectedLevels.has(entry.level)) {
          return false;
        }
      }
      return true;
    });
  }, [logs, selectedLevels]);

  // Find all matching log indices for search navigation
  const searchMatches = useMemo(() => {
    if (!searchText) return [];
    const matches: number[] = [];
    filteredLogs.forEach((entry, index) => {
      const message = (entry.message || entry.raw || "").toLowerCase();
      if (message.includes(searchText.toLowerCase())) {
        matches.push(index);
      }
    });
    return matches;
  }, [filteredLogs, searchText]);

  // Reset current match index when search changes
  // biome-ignore lint/correctness/useExhaustiveDependencies: intentionally reset when searchText changes
  useEffect(() => {
    setCurrentMatchIndex(0);
  }, [searchText]);

  const availableLogLevels = useMemo(() => {
    const levels = new Set<LogLevel>();
    logs.forEach((entry) => {
      if (entry.level) {
        levels.add(entry.level);
      }
    });
    return Array.from(levels).sort();
  }, [logs]);

  // Virtualization setup (must be before navigation functions)
  const rowVirtualizer = useVirtualizer({
    count: filteredLogs.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => (wrapText ? 60 : 36),
    overscan: 5,
  });

  // Navigate to previous match
  const goToPreviousMatch = useCallback(() => {
    if (searchMatches.length === 0) return;
    const newIndex = currentMatchIndex > 0 ? currentMatchIndex - 1 : searchMatches.length - 1;
    setCurrentMatchIndex(newIndex);
    rowVirtualizer.scrollToIndex(searchMatches[newIndex], { align: "center" });
  }, [searchMatches, currentMatchIndex, rowVirtualizer]);

  // Navigate to next match
  const goToNextMatch = useCallback(() => {
    if (searchMatches.length === 0) return;
    const newIndex = currentMatchIndex < searchMatches.length - 1 ? currentMatchIndex + 1 : 0;
    setCurrentMatchIndex(newIndex);
    rowVirtualizer.scrollToIndex(searchMatches[newIndex], { align: "center" });
  }, [searchMatches, currentMatchIndex, rowVirtualizer]);

  // Helper to highlight search text in message
  const highlightSearchText = useCallback((text: string, isCurrentMatch: boolean): React.ReactNode => {
    if (!searchText || !text) return text;

    const lowerText = text.toLowerCase();
    const lowerSearch = searchText.toLowerCase();
    const index = lowerText.indexOf(lowerSearch);

    if (index === -1) return text;

    const before = text.slice(0, index);
    const match = text.slice(index, index + searchText.length);
    const after = text.slice(index + searchText.length);

    return (
      <>
        {before}
        <mark className={`px-0.5 rounded ${isCurrentMatch ? "bg-yellow-400 dark:bg-yellow-500" : "bg-yellow-200 dark:bg-yellow-700"}`}>
          {match}
        </mark>
        {highlightSearchText(after, isCurrentMatch)}
      </>
    );
  }, [searchText]);

  return (
    <div className="min-h-screen bg-background">
      <div className="container mx-auto px-4 py-6 max-w-7xl">
        <div className="space-y-6">
          {/* Header */}
          <div className="flex items-center gap-4">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() =>
                    navigate({
                      to: "/",
                    })
                  }
                >
                  <ArrowLeftIcon className="size-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Back to Dashboard</TooltipContent>
            </Tooltip>
            <div className="flex-1">
              <h1 className="text-2xl font-bold">Container Logs</h1>
              {container && (
                <p className="text-sm text-muted-foreground">
                  {container.names?.[0]?.replace(/^\//, "") ||
                    containerIdentifier}
                </p>
              )}
            </div>
          </div>

          {/* Container Info Card */}
          {container && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Container Details</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                    <div>
                      <span className="text-muted-foreground block mb-1">
                        Name
                      </span>
                      <p className="font-medium">
                        {formatContainerName(container.names)}
                      </p>
                    </div>
                    <div className="md:col-span-2">
                      <span className="text-muted-foreground block mb-1">
                        ID
                      </span>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <p className="font-mono text-xs truncate cursor-help">
                            {container.id}
                          </p>
                        </TooltipTrigger>
                        <TooltipContent className="max-w-md">
                          {container.id}
                        </TooltipContent>
                      </Tooltip>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                    <div>
                      <span className="text-muted-foreground block mb-1">
                        Image
                      </span>
                      <p className="font-medium">{container.image}</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground block mb-1">
                        State
                      </span>
                      <Badge
                        className={`${getStateBadgeClass(container.state)} border-0`}
                      >
                        {toTitleCase(container.state)}
                      </Badge>
                    </div>
                    <div>
                      <span className="text-muted-foreground block mb-1">
                        Status
                      </span>
                      <p className="font-medium">{container.status}</p>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                    <div>
                      <span className="text-muted-foreground block mb-1">
                        Uptime
                      </span>
                      <p className="font-medium">
                        {formatUptime(container.created)}
                      </p>
                    </div>
                    <div>
                      <span className="text-muted-foreground block mb-1">
                        Created
                      </span>
                      <p className="font-medium">
                        {formatCreatedDate(container.created)}
                      </p>
                    </div>
                    <div>
                      <span className="text-muted-foreground block mb-1">
                        Command
                      </span>
                      <p className="font-mono text-xs break-all">
                        {container.command}
                      </p>
                    </div>
                  </div>

                  {/* Labels Section */}
                  {container.labels &&
                    Object.keys(container.labels).length > 0 && (
                      <div className="space-y-2 border-t pt-4">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setShowLabels((value) => !value)}
                          className="h-8 w-full justify-start text-muted-foreground hover:text-foreground"
                        >
                          <ChevronDownIcon
                            className={`mr-2 size-4 transition-transform ${
                              showLabels ? "rotate-180" : ""
                            }`}
                          />
                          {showLabels ? "Hide" : "Show"} container labels (
                          {Object.keys(container.labels).length})
                        </Button>
                        {showLabels && (
                          <div className="max-h-[200px] space-y-2 overflow-y-auto rounded-md border bg-muted/30 p-3">
                            {Object.entries(container.labels).map(
                              ([key, value]) => (
                                <div
                                  key={key}
                                  className="rounded-md bg-background p-2 text-xs"
                                >
                                  <div className="mb-1 font-semibold text-foreground">
                                    {key}
                                  </div>
                                  <div className="break-all font-mono text-muted-foreground">
                                    {value}
                                  </div>
                                </div>
                              )
                            )}
                          </div>
                        )}
                      </div>
                    )}

                  {/* Environment Variables Section */}
                  <div className="space-y-2 border-t pt-4">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowEnvVariables((value) => !value)}
                      className="h-8 w-full justify-start text-muted-foreground hover:text-foreground"
                    >
                      <ChevronDownIcon
                        className={`mr-2 size-4 transition-transform ${
                          showEnvVariables ? "rotate-180" : ""
                        }`}
                      />
                      {showEnvVariables ? "Hide" : "Show"} environment variables
                    </Button>
                    {showEnvVariables && container && (
                      <div className="max-h-[300px] overflow-y-auto">
                        <EnvironmentVariables
                          containerId={actualContainerId}
                          containerHost={container.host}
                          onContainerIdChange={handleContainerRecreated}
                        />
                      </div>
                    )}
                  </div>

                  {/* Terminal Section */}
                  <div className="space-y-2 border-t pt-4">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowTerminal((value) => !value)}
                      className="h-8 w-full justify-start text-muted-foreground hover:text-foreground"
                    >
                      <ChevronDownIcon
                        className={`mr-2 size-4 transition-transform ${
                          showTerminal ? "rotate-180" : ""
                        }`}
                      />
                      <TerminalIcon className="mr-2 size-4" />
                      {showTerminal ? "Hide" : "Show"} terminal
                    </Button>
                    {container && (
                      <div className={`mt-2 ${showTerminal ? "" : "hidden"}`}>
                        <Terminal
                          containerId={actualContainerId}
                          host={container.host}
                        />
                      </div>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          )}

          {/* Logs Card */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">
                  Logs
                  {filteredLogs.length !== logs.length && (
                    <span className="ml-2 text-xs text-muted-foreground font-normal">
                      ({filteredLogs.length} of {logs.length})
                    </span>
                  )}
                </CardTitle>
                <div className="flex items-center gap-2">
                  <div className="flex items-center gap-2">
                    <Label
                      htmlFor={logLinesInputId}
                      className="text-xs text-muted-foreground"
                    >
                      Lines
                    </Label>
                    <Input
                      id={logLinesInputId}
                      type="number"
                      min="1"
                      value={logLines}
                      onChange={(e) => handleLogLinesChange(e.target.value)}
                      disabled={isStreaming}
                      className="h-8 w-20 text-xs"
                    />
                  </div>
                  <Button
                    variant={isStreaming ? "default" : "outline"}
                    size="sm"
                    onClick={handleToggleStream}
                    disabled={isLoadingLogs && !isStreaming}
                  >
                    {isStreaming ? (
                      <>
                        <SquareIcon className="mr-2 size-4" />
                        Stop
                      </>
                    ) : (
                      <>
                        <PlayIcon className="mr-2 size-4" />
                        Stream
                      </>
                    )}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleRefresh}
                    disabled={isStreaming || isLoadingLogs}
                  >
                    <RefreshCcwIcon className="mr-2 size-4" />
                  </Button>
                </div>
              </div>

              {/* Search and Filter Controls */}
              <div className="flex flex-wrap items-center gap-2 pt-4">
                <div className="relative flex-1 min-w-[200px]">
                  <SearchIcon className="absolute left-2 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
                  <Input
                    placeholder="Search logs..."
                    value={searchText}
                    onChange={(e) => setSearchText(e.target.value)}
                    className="pl-8 h-9 text-xs"
                  />
                </div>

                {/* Search navigation controls */}
                {searchText && (
                  <div className="flex items-center gap-1">
                    <span className="text-xs text-muted-foreground whitespace-nowrap">
                      {searchMatches.length > 0
                        ? `${currentMatchIndex + 1} of ${searchMatches.length}`
                        : "No matches"}
                    </span>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={goToPreviousMatch}
                          disabled={searchMatches.length === 0}
                          className="h-9 w-9 p-0"
                        >
                          <ChevronLeftIcon className="size-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>Previous match</TooltipContent>
                    </Tooltip>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={goToNextMatch}
                          disabled={searchMatches.length === 0}
                          className="h-9 w-9 p-0"
                        >
                          <ChevronRightIcon className="size-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>Next match</TooltipContent>
                    </Tooltip>
                  </div>
                )}

                <Popover open={showFilters} onOpenChange={setShowFilters}>
                  <PopoverTrigger asChild>
                    <Button variant="outline" size="sm" className="h-9">
                      <FilterIcon className="mr-2 size-4" />
                      Filter
                      {selectedLevels.size > 0 && (
                        <Badge
                          variant="outline"
                          className="ml-2 px-1 py-0 h-4 text-xs"
                        >
                          {selectedLevels.size}
                        </Badge>
                      )}
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent align="end" className="w-56">
                    <div className="space-y-3">
                      <div>
                        <h4 className="text-sm font-medium mb-2">Log Levels</h4>
                        <div className="space-y-2">
                          {availableLogLevels.length === 0 ? (
                            <p className="text-xs text-muted-foreground">
                              No log levels available
                            </p>
                          ) : (
                            availableLogLevels.map((level) => (
                              <label
                                key={level}
                                className="flex items-center gap-2 cursor-pointer"
                              >
                                <button
                                  type="button"
                                  onClick={() => toggleLogLevel(level)}
                                  className={`size-4 rounded border flex items-center justify-center ${
                                    selectedLevels.has(level)
                                      ? "bg-primary border-primary"
                                      : "border-input"
                                  }`}
                                >
                                  {selectedLevels.has(level) && (
                                    <CheckIcon className="size-3 text-primary-foreground" />
                                  )}
                                </button>
                                <Badge
                                  variant="outline"
                                  className={`text-xs ${getLogLevelBadgeColor(level)}`}
                                >
                                  {level}
                                </Badge>
                              </label>
                            ))
                          )}
                        </div>
                      </div>
                      {selectedLevels.size > 0 && (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setSelectedLevels(new Set())}
                          className="w-full"
                        >
                          Clear Filters
                        </Button>
                      )}
                    </div>
                  </PopoverContent>
                </Popover>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setShowTimestamps(!showTimestamps)}
                      className="h-9"
                    >
                      {showTimestamps ? (
                        <EyeIcon className="size-4" />
                      ) : (
                        <EyeOffIcon className="size-4" />
                      )}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {showTimestamps ? "Hide" : "Show"} timestamps
                  </TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant={autoScroll ? "secondary" : "outline"}
                      size="sm"
                      onClick={() => setAutoScroll(!autoScroll)}
                      className={`h-9 ${autoScroll ? "bg-primary/10 hover:bg-primary/20 border-primary/30" : ""}`}
                    >
                      {autoScroll ? (
                        <ArrowDownToLineIcon className="size-4" />
                      ) : (
                        <ArrowDownIcon className="size-4" />
                      )}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    Auto-scroll: {autoScroll ? "On" : "Off"}
                  </TooltipContent>
                </Tooltip>

                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setWrapText(!wrapText)}
                      className="h-9"
                    >
                      <WrapTextIcon
                        className={`size-4 ${wrapText ? "text-primary" : ""}`}
                      />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    Text wrap: {wrapText ? "On" : "Off"}
                  </TooltipContent>
                </Tooltip>

                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" size="sm" className="h-9">
                      <DownloadIcon className="mr-2 size-4" />
                      Download
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem
                      onClick={() => handleDownloadLogs("json")}
                    >
                      Download as JSON
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => handleDownloadLogs("txt")}>
                      Download as TXT
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              <div
                ref={parentRef}
                className="h-[calc(100vh-400px)] min-h-[400px] w-full overflow-auto"
              >
                {isLoadingLogs && logs.length === 0 ? (
                  <div className="flex items-center justify-center py-8 text-muted-foreground">
                    <Spinner className="mr-2 size-4" />
                    Loading logs...
                  </div>
                ) : logs.length === 0 ? (
                  <div className="flex items-center justify-center py-8 text-muted-foreground text-sm">
                    No logs available
                  </div>
                ) : filteredLogs.length === 0 ? (
                  <div className="flex items-center justify-center py-8 text-muted-foreground text-sm">
                    No logs match the current filters
                  </div>
                ) : (
                  <div
                    style={{
                      height: `${rowVirtualizer.getTotalSize()}px`,
                      width: "100%",
                      position: "relative",
                    }}
                    className={`font-mono text-xs ${wrapText ? "" : "w-fit min-w-full"}`}
                  >
                    {rowVirtualizer.getVirtualItems().map((virtualRow) => {
                      const entry = filteredLogs[virtualRow.index];
                      if (!entry.message?.trim()) return null;

                      const timestamp = entry.timestamp
                        ? new Date(entry.timestamp)
                        : null;
                      const dateLabel = timestamp
                        ? `${timestamp.toLocaleDateString("en-GB", {
                            day: "2-digit",
                            month: "2-digit",
                            year: "numeric",
                          })} ${timestamp.toLocaleTimeString("en-US", {
                            hour12: false,
                            hour: "2-digit",
                            minute: "2-digit",
                            second: "2-digit",
                          })}`
                        : "—";

                      // Check if this row is the current search match
                      const isCurrentMatch = searchMatches.length > 0 && searchMatches[currentMatchIndex] === virtualRow.index;
                      const hasMatch = searchMatches.includes(virtualRow.index);

                      return (
                        <div
                          key={virtualRow.key}
                          data-index={virtualRow.index}
                          ref={rowVirtualizer.measureElement}
                          style={{
                            position: "absolute",
                            top: 0,
                            left: 0,
                            width: wrapText ? "100%" : "max-content",
                            minWidth: "100%",
                            transform: `translateY(${virtualRow.start}px)`,
                          }}
                          className={`group flex items-start gap-3 px-4 py-1.5 hover:bg-muted/50 ${
                            wrapText ? "" : "whitespace-nowrap"
                          } ${isCurrentMatch ? "bg-yellow-100 dark:bg-yellow-900/30 border-y-2 border-yellow-400 dark:border-yellow-600" : virtualRow.index % 2 === 0 ? "bg-muted/30" : ""}`}
                        >
                          {showTimestamps && (
                            <span className="text-muted-foreground shrink-0 text-[11px]">
                              {dateLabel}
                            </span>
                          )}
                          <Badge
                            variant="outline"
                            className={`shrink-0 text-xs px-1.5 py-0 h-5 ${getLogLevelBadgeColor(entry.level ?? "UNKNOWN")}`}
                          >
                            {entry.level ?? "UNKNOWN"}
                          </Badge>
                          <span
                            className={`text-foreground flex-1 ${wrapText ? "break-words" : ""}`}
                          >
                            {hasMatch
                              ? highlightSearchText(entry.message ?? "", isCurrentMatch)
                              : entry.message ?? ""}
                          </span>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <button
                                type="button"
                                onClick={() => handleCopyLog(entry)}
                                className="shrink-0 opacity-0 group-hover:opacity-100 transition-opacity p-1 hover:bg-muted rounded"
                              >
                                <CopyIcon className="size-3 text-muted-foreground" />
                              </button>
                            </TooltipTrigger>
                            <TooltipContent>Copy log entry</TooltipContent>
                          </Tooltip>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
