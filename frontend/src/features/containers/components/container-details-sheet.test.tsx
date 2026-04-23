import type { ReactNode } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { ContainerDetailsSheet } from "./container-details-sheet";

const mockUseContainerStats = vi.fn();
const mockUseContainerHistory = vi.fn();

vi.mock("../hooks/use-container-stats", () => ({
  useContainerStats: (...args: unknown[]) => mockUseContainerStats(...args),
}));

vi.mock("../hooks/use-container-history", () => ({
  useContainerHistory: (...args: unknown[]) => mockUseContainerHistory(...args),
}));

vi.mock("@/components/ui/tabs", async () => {
  const React = await import("react");
  const TabsContext = React.createContext<{ value: string; onValueChange: (v: string) => void }>({ value: "", onValueChange: () => {} });
  
  return {
    Tabs: ({ value, defaultValue, onValueChange, children }: any) => {
      const [v, setV] = React.useState(value || defaultValue);
      const handleChange = (newV: string) => {
        setV(newV);
        if (onValueChange) onValueChange(newV);
      };
      React.useEffect(() => { if (value !== undefined) setV(value); }, [value]);
      return <TabsContext.Provider value={{ value: v, onValueChange: handleChange }}><div>{children}</div></TabsContext.Provider>;
    },
    TabsContent: ({ value, children }: any) => {
      const context = React.useContext(TabsContext);
      if (context.value !== value) return null;
      return <div>{children}</div>;
    },
    TabsList: ({ children }: any) => <div>{children}</div>,
    TabsTrigger: ({ value, children, disabled, ...props }: any) => {
      const context = React.useContext(TabsContext);
      return <button type="button" disabled={disabled} onClick={() => context.onValueChange(value)} {...props}>{children}</button>;
    },
  };
});

vi.mock("./environment-variables", () => ({
  EnvironmentVariables: ({
    onContainerIdChange,
  }: {
    onContainerIdChange: (containerId: string) => void;
  }) => (
    <button type="button" onClick={() => onContainerIdChange("container-2")}>
      Update container id
    </button>
  ),
}));

vi.mock("recharts", async () => {
  const actual = await vi.importActual<typeof import("recharts")>("recharts");

  return {
    ...actual,
    ResponsiveContainer: ({ children }: { children?: ReactNode }) => (
      <div style={{ width: 960, height: 320 }}>{children}</div>
    ),
  };
});

beforeAll(() => {
  class ResizeObserverMock {
    disconnect() {}
    observe() {}
    unobserve() {}
  }

  vi.stubGlobal("ResizeObserver", ResizeObserverMock);
  Object.defineProperty(HTMLElement.prototype, "clientWidth", {
    configurable: true,
    value: 960,
  });
  Object.defineProperty(HTMLElement.prototype, "clientHeight", {
    configurable: true,
    value: 320,
  });
});

describe("ContainerDetailsSheet", () => {
  beforeEach(() => {
    vi.clearAllMocks();

    mockUseContainerStats.mockReturnValue({
      stats: null,
      history: [],
      isConnected: false,
      error: null,
      connect: vi.fn(),
      disconnect: vi.fn(),
      clearHistory: vi.fn(),
    });

    mockUseContainerHistory.mockReturnValue({
      data: {
        cpu_1h: 12,
        memory_1h: 34,
        cpu_12h: 18,
        memory_12h: 40,
        has_data: true,
        samples: [
          {
            container_id: "container-1",
            host: "local",
            cpu_percent: 10,
            memory_usage: 512,
            memory_limit: 1024,
            memory_percent: 50,
            network_rx: 100,
            network_tx: 80,
            block_read: 20,
            block_write: 10,
            pids: 4,
            timestamp: 1_700_000_000,
          },
          {
            container_id: "container-1",
            host: "local",
            cpu_percent: 14,
            memory_usage: 600,
            memory_limit: 1024,
            memory_percent: 58,
            network_rx: 120,
            network_tx: 95,
            block_read: 25,
            block_write: 12,
            pids: 5,
            timestamp: 1_700_000_030,
          },
        ],
      },
    });
  });

  it("renders stats charts from persisted samples before live history arrives", () => {
    render(
      <ContainerDetailsSheet
        container={{
          id: "container-1",
          names: ["/api"],
          image: "ghcr.io/example/api:latest",
          image_id: "sha256:123",
          command: "node server.js",
          created: 1_700_000_000,
          state: "running",
          status: "Up 2 hours",
          host: "local",
          historical_stats: {
            cpu_1h: 12,
            memory_1h: 34,
            cpu_12h: 18,
            memory_12h: 40,
          },
        }}
        host="local"
        isOpen
        onOpenChange={vi.fn()}
      />,
    );

    expect(screen.getByText("CPU Usage (%)")).toBeInTheDocument();
    expect(screen.getByText("Memory Usage (%)")).toBeInTheDocument();
    expect(screen.getByText("Network I/O")).toBeInTheDocument();
    expect(screen.getByText("Block I/O")).toBeInTheDocument();
  });

  it("keeps hooks stable with a null container and respects child container id updates", async () => {
    const { rerender } = render(
      <ContainerDetailsSheet
        container={null}
        host="local"
        isOpen
        onOpenChange={vi.fn()}
      />,
    );

    rerender(
      <ContainerDetailsSheet
        container={{
          id: "container-1",
          names: ["/api"],
          image: "ghcr.io/example/api:latest",
          image_id: "sha256:123",
          command: "node server.js",
          created: 1_700_000_000,
          state: "running",
          status: "Up 2 hours",
          host: "local",
        }}
        host="local"
        isOpen
        onOpenChange={vi.fn()}
      />,
    );

    fireEvent.click(
      await screen.findByRole("button", { name: /update container id/i }),
    );

    expect(mockUseContainerStats).toHaveBeenLastCalledWith(
      expect.objectContaining({ containerId: "container-2" }),
    );
  });
});
