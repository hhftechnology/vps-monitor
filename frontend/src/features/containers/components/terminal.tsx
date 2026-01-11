import "xterm/css/xterm.css";

import { ArrowDownIcon, CopyIcon, RefreshCwIcon } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { Terminal as XTerm } from "xterm";
import { FitAddon } from "xterm-addon-fit";

import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { getAuthToken } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

interface TerminalProps {
  containerId: string;
  host: string;
}

const TERMINAL_THEME = {
  background: "#ffffff",
  foreground: "#1a1a1a",
  cursor: "#2563eb",
  cursorAccent: "#ffffff",
  selectionBackground: "rgba(37, 99, 235, 0.15)",
  selectionInactiveBackground: "rgba(0, 0, 0, 0.08)",
  black: "#000000",
  red: "#dc2626",
  green: "#16a34a",
  yellow: "#ca8a04",
  blue: "#2563eb",
  magenta: "#9333ea",
  cyan: "#0891b2",
  white: "#6b7280",
  brightBlack: "#374151",
  brightRed: "#ef4444",
  brightGreen: "#22c55e",
  brightYellow: "#eab308",
  brightBlue: "#3b82f6",
  brightMagenta: "#a855f7",
  brightCyan: "#06b6d4",
  brightWhite: "#1f2937",
} as const;

const createTerminal = () => {
  const term = new XTerm({
    cursorBlink: true,
    cursorStyle: "bar",
    cursorWidth: 5,
    scrollback: 10000,
    fastScrollModifier: "shift",
    fastScrollSensitivity: 5,
    theme: TERMINAL_THEME,
    fontFamily:
      '"Google Sans Code", "PT Mono", Menlo, Monaco, "Courier New", monospace',
    fontSize: 14,
    lineHeight: 1.5,
    letterSpacing: 0.5,
    allowProposedApi: true,
  });

  const fitAddon = new FitAddon();
  term.loadAddon(fitAddon);

  return { term, fitAddon };
};

const buildWebSocketUrl = (containerId: string, host: string) => {
  let wsHost: string;
  let wsProtocol: string;

  if (API_BASE_URL) {
    const apiUrl = new URL(API_BASE_URL, window.location.href);
    wsHost = apiUrl.host;
    wsProtocol = apiUrl.protocol === "https:" ? "wss:" : "ws:";
  } else {
    wsHost = window.location.host;
    wsProtocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  }

  const token = getAuthToken();
  const params = new URLSearchParams({ host });
  if (token) {
    params.set("token", token);
  }

  return `${wsProtocol}//${wsHost}/api/v1/containers/${containerId}/exec?${params.toString()}`;
};

export function Terminal({ containerId, host }: TerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const cleanupRef = useRef<(() => void) | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [isReconnecting, setIsReconnecting] = useState(false);

  const handleCopyTerminal = () => {
    if (!xtermRef.current) return;

    const selection = xtermRef.current.getSelection();
    if (selection) {
      navigator.clipboard.writeText(selection);
      toast.success("Selection copied to clipboard");
      return;
    }

    xtermRef.current.selectAll();
    const allContent = xtermRef.current.getSelection();
    xtermRef.current.clearSelection();
    if (allContent) {
      navigator.clipboard.writeText(allContent);
      toast.success("Terminal content copied to clipboard");
    }
  };

  const handleScrollToBottom = () => {
    if (!xtermRef.current) return;
    const buffer = xtermRef.current.buffer.active;
    xtermRef.current.scrollToLine(buffer.baseY + buffer.cursorY);
  };

  const connect = useCallback(() => {
    if (!terminalRef.current) return;

    // Clean up existing connection if any
    if (cleanupRef.current) {
      cleanupRef.current();
      cleanupRef.current = null;
    }

    setIsReconnecting(true);

    // Create or reuse terminal
    let term = xtermRef.current;
    let fitAddon = fitAddonRef.current;

    if (!term || !fitAddon) {
      const created = createTerminal();
      term = created.term;
      fitAddon = created.fitAddon;

      term.open(terminalRef.current);
      fitAddon.fit();

      xtermRef.current = term;
      fitAddonRef.current = fitAddon;
    } else {
      // Clear existing content for reconnection
      term.clear();
    }

    const focusTimeout = window.setTimeout(() => {
      term.focus();
    }, 100);

    const ws = new WebSocket(buildWebSocketUrl(containerId, host));
    ws.binaryType = "arraybuffer";
    wsRef.current = ws;

    const sendResize = () => {
      if (!terminalRef.current) return;
      if (ws.readyState !== WebSocket.OPEN) return;

      fitAddon.fit();
      const { cols, rows } = term;
      ws.send(JSON.stringify({ type: "resize", cols, rows }));
    };

    ws.onopen = () => {
      setIsConnected(true);
      setIsReconnecting(false);
      term.write(
        "\r\n\x1b[32m✓ Connected to container terminal\x1b[0m\r\n\r\n"
      );
      sendResize();
      term.scrollToBottom();
    };

    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        term.write(new Uint8Array(event.data));
      } else if (typeof event.data === "string") {
        term.write(event.data);
      }
      term.scrollToBottom();
    };

    ws.onclose = () => {
      setIsConnected(false);
      setIsReconnecting(false);
      term.write("\r\n\x1b[31m✗ Connection closed\x1b[0m\r\n");
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      setIsConnected(false);
      setIsReconnecting(false);
      term.write("\r\n\x1b[31m✗ WebSocket error\x1b[0m\r\n");
    };

    const dataSubscription = term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data);
        term.scrollToBottom();
      }
    });

    window.addEventListener("resize", sendResize);
    const resizeObserver =
      typeof ResizeObserver !== "undefined"
        ? new ResizeObserver(() => {
            sendResize();
          })
        : null;

    if (resizeObserver && terminalRef.current) {
      resizeObserver.observe(terminalRef.current);
    }

    const resizeTimeout = window.setTimeout(() => {
      sendResize();
    }, 100);

    // Store cleanup function
    cleanupRef.current = () => {
      window.removeEventListener("resize", sendResize);
      resizeObserver?.disconnect();
      window.clearTimeout(resizeTimeout);
      window.clearTimeout(focusTimeout);
      dataSubscription.dispose();
      if (
        ws.readyState === WebSocket.OPEN ||
        ws.readyState === WebSocket.CONNECTING
      ) {
        ws.close();
      }
    };
  }, [containerId, host]);

  const handleReconnect = () => {
    if (isReconnecting) return;
    toast.info("Reconnecting to terminal...");
    connect();
  };

  useEffect(() => {
    connect();

    return () => {
      if (cleanupRef.current) {
        cleanupRef.current();
        cleanupRef.current = null;
      }
      if (xtermRef.current) {
        xtermRef.current.dispose();
        xtermRef.current = null;
        fitAddonRef.current = null;
      }
    };
  }, [connect]);

  return (
    <div className="w-full space-y-2">
      <div className="flex items-center justify-between px-3 py-2 bg-muted/30 rounded-t-md border border-b-0 border-border">
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5">
            <div
              className={`size-2 rounded-full ${isConnected ? "bg-green-500 animate-pulse" : "bg-red-500"}`}
            />
            <span className="text-xs font-medium text-muted-foreground">
              {isConnected ? "Connected" : "Disconnected"}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-1">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleReconnect}
                disabled={isReconnecting || isConnected}
                className="h-7 px-2"
              >
                <RefreshCwIcon
                  className={`size-3.5 ${isReconnecting ? "animate-spin" : ""}`}
                />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              {isConnected
                ? "Connected"
                : isReconnecting
                  ? "Reconnecting..."
                  : "Reconnect"}
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleScrollToBottom}
                className="h-7 px-2"
              >
                <ArrowDownIcon className="size-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Scroll to bottom</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                onClick={handleCopyTerminal}
                className="h-7 px-2"
              >
                <CopyIcon className="size-3.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Copy terminal content</TooltipContent>
          </Tooltip>
        </div>
      </div>

      <div
        ref={terminalRef}
        className="w-full h-[400px] rounded-b-md overflow-hidden border border-t-0 border-border bg-white shadow-sm p-2 pb-4"
      />
    </div>
  );
}
