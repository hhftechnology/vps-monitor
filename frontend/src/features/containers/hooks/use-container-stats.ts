import { useCallback, useEffect, useRef, useState } from "react";

import { getAuthToken } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { ContainerStats } from "../types/stats";

const MAX_HISTORY_LENGTH = 60; // Keep last 60 data points for graphs

interface UseContainerStatsOptions {
  containerId: string;
  host: string;
  enabled?: boolean;
  historyLength?: number;
}

interface UseContainerStatsReturn {
  stats: ContainerStats | null;
  history: ContainerStats[];
  isConnected: boolean;
  error: string | null;
  connect: () => void;
  disconnect: () => void;
  clearHistory: () => void;
}

export function useContainerStats({
  containerId,
  host,
  enabled = true,
  historyLength = MAX_HISTORY_LENGTH,
}: UseContainerStatsOptions): UseContainerStatsReturn {
  const [stats, setStats] = useState<ContainerStats | null>(null);
  const [history, setHistory] = useState<ContainerStats[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const baseUrl = API_BASE_URL || window.location.origin;
    const wsBase = baseUrl.replace(/^https?:/, protocol);
    const token = getAuthToken();
    const tokenParam = token ? `&token=${encodeURIComponent(token)}` : "";

    const wsUrl = `${wsBase}/api/v1/containers/${encodeURIComponent(containerId)}/stats?host=${encodeURIComponent(host)}${tokenParam}`;

    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      setIsConnected(true);
      setError(null);
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as ContainerStats;
        setStats(data);
        setHistory((prev) => {
          const updated = [...prev, data];
          // Keep only the last N data points
          if (updated.length > historyLength) {
            return updated.slice(-historyLength);
          }
          return updated;
        });
      } catch {
        console.error("Failed to parse stats data");
      }
    };

    ws.onerror = () => {
      setError("WebSocket connection error");
      setIsConnected(false);
    };

    ws.onclose = () => {
      setIsConnected(false);
    };

    wsRef.current = ws;
  }, [containerId, host, historyLength]);

  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsConnected(false);
  }, []);

  const clearHistory = useCallback(() => {
    setHistory([]);
  }, []);

  useEffect(() => {
    if (enabled && containerId && host) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [enabled, containerId, host, connect, disconnect]);

  return {
    stats,
    history,
    isConnected,
    error,
    connect,
    disconnect,
    clearHistory,
  };
}
