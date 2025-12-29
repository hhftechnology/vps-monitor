import { useCallback, useEffect, useRef, useState } from "react";

import { getAuthToken } from "@/lib/api-client";
import { API_BASE_URL } from "@/types/api";

import type { ContainerStats } from "../types/stats";

interface UseContainerStatsOptions {
  containerId: string;
  host: string;
  enabled?: boolean;
}

interface UseContainerStatsReturn {
  stats: ContainerStats | null;
  isConnected: boolean;
  error: string | null;
  connect: () => void;
  disconnect: () => void;
}

export function useContainerStats({
  containerId,
  host,
  enabled = true,
}: UseContainerStatsOptions): UseContainerStatsReturn {
  const [stats, setStats] = useState<ContainerStats | null>(null);
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
  }, [containerId, host]);

  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsConnected(false);
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
    isConnected,
    error,
    connect,
    disconnect,
  };
}
