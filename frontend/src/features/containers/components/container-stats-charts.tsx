import { useMemo } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  XAxis,
  YAxis,
} from "recharts";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";

import type { ContainerStats } from "../types/stats";

interface ContainerStatsChartsProps {
  history: ContainerStats[];
}

type ChartData = {
  time: string;
  timestamp: number;
  cpu: number;
  memory: number;
  memoryUsage: number;
  memoryLimit: number;
  networkRx: number;
  networkTx: number;
  blockRead: number;
  blockWrite: number;
};

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";

  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return `${(bytes / k ** i).toFixed(1)} ${sizes[i]}`;
}

function formatTime(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

const cpuChartConfig = {
  cpu: {
    label: "CPU",
    color: "var(--chart-1)",
  },
} satisfies ChartConfig;

const memoryChartConfig = {
  memory: {
    label: "Memory",
    color: "var(--chart-2)",
  },
} satisfies ChartConfig;

const networkChartConfig = {
  networkRx: {
    label: "RX (Received)",
    color: "var(--chart-3)",
  },
  networkTx: {
    label: "TX (Transmitted)",
    color: "var(--chart-4)",
  },
} satisfies ChartConfig;

const blockChartConfig = {
  blockRead: {
    label: "Read",
    color: "var(--chart-5)",
  },
  blockWrite: {
    label: "Write",
    color: "var(--destructive)",
  },
} satisfies ChartConfig;

interface StatsChartCardProps {
  title: string;
  data: ChartData[];
  config: ChartConfig;
  children: React.ReactElement;
  legend?: React.ReactNode;
}

function StatsChartCard({
  title,
  data,
  config,
  children,
  legend,
}: StatsChartCardProps) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <ChartContainer config={config} className="h-[220px] w-full">
          {children}
        </ChartContainer>
        {data.length > 0 ? legend : null}
      </CardContent>
    </Card>
  );
}

function SeriesLegend({ config }: { config: ChartConfig }) {
  return (
    <div className="flex flex-wrap justify-center gap-4 text-xs text-muted-foreground">
      {Object.entries(config).map(([key, value]) => (
        <div key={key} className="flex items-center gap-1.5">
          <span
            className="size-2 rounded-full"
            style={{ backgroundColor: value.color }}
          />
          <span>{value.label}</span>
        </div>
      ))}
    </div>
  );
}

export function ContainerStatsCharts({ history }: ContainerStatsChartsProps) {
  const chartData = useMemo<ChartData[]>(() => {
    return history.map((stat) => ({
      time: formatTime(stat.timestamp),
      timestamp: stat.timestamp,
      cpu: Number.parseFloat(stat.cpu_percent.toFixed(2)),
      memory: Number.parseFloat(stat.memory_percent.toFixed(2)),
      memoryUsage: stat.memory_usage,
      memoryLimit: stat.memory_limit,
      networkRx: stat.network_rx,
      networkTx: stat.network_tx,
      blockRead: stat.block_read,
      blockWrite: stat.block_write,
    }));
  }, [history]);

  if (chartData.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-muted-foreground">
        No data yet. Stats will appear once streaming begins.
      </div>
    );
  }

  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <StatsChartCard title="CPU Usage (%)" data={chartData} config={cpuChartConfig}>
        <LineChart accessibilityLayer data={chartData}>
          <CartesianGrid vertical={false} />
          <XAxis
            axisLine={false}
            dataKey="time"
            minTickGap={32}
            tickLine={false}
          />
          <YAxis
            axisLine={false}
            domain={[0, 100]}
            tickFormatter={(value) => `${value}%`}
            tickLine={false}
            width={44}
          />
          <ChartTooltip
            content={
              <ChartTooltipContent
                formatter={(value) => `${Number(value).toFixed(2)}%`}
              />
            }
          />
          <Line
            dataKey="cpu"
            dot={false}
            stroke="var(--color-cpu)"
            strokeWidth={2}
            type="monotone"
          />
        </LineChart>
      </StatsChartCard>

      <StatsChartCard
        title="Memory Usage (%)"
        data={chartData}
        config={memoryChartConfig}
      >
        <LineChart accessibilityLayer data={chartData}>
          <CartesianGrid vertical={false} />
          <XAxis
            axisLine={false}
            dataKey="time"
            minTickGap={32}
            tickLine={false}
          />
          <YAxis
            axisLine={false}
            domain={[0, 100]}
            tickFormatter={(value) => `${value}%`}
            tickLine={false}
            width={44}
          />
          <ChartTooltip
            content={
              <ChartTooltipContent
                formatter={(value, _name, item) => {
                  const payload = item.payload as ChartData | undefined;
                  return `${Number(value).toFixed(2)}% (${formatBytes(payload?.memoryUsage ?? 0)} / ${formatBytes(payload?.memoryLimit ?? 0)})`;
                }}
              />
            }
          />
          <Line
            dataKey="memory"
            dot={false}
            stroke="var(--color-memory)"
            strokeWidth={2}
            type="monotone"
          />
        </LineChart>
      </StatsChartCard>

      <StatsChartCard
        title="Network I/O"
        data={chartData}
        config={networkChartConfig}
        legend={<SeriesLegend config={networkChartConfig} />}
      >
        <LineChart accessibilityLayer data={chartData}>
          <CartesianGrid vertical={false} />
          <XAxis
            axisLine={false}
            dataKey="time"
            minTickGap={32}
            tickLine={false}
          />
          <YAxis
            axisLine={false}
            tickFormatter={(value) => formatBytes(Number(value))}
            tickLine={false}
            width={64}
          />
          <ChartTooltip
            content={
              <ChartTooltipContent
                formatter={(value) => formatBytes(Number(value))}
              />
            }
          />
          <Line
            dataKey="networkRx"
            dot={false}
            stroke="var(--color-networkRx)"
            strokeWidth={2}
            type="monotone"
          />
          <Line
            dataKey="networkTx"
            dot={false}
            stroke="var(--color-networkTx)"
            strokeWidth={2}
            type="monotone"
          />
        </LineChart>
      </StatsChartCard>

      <StatsChartCard
        title="Block I/O"
        data={chartData}
        config={blockChartConfig}
        legend={<SeriesLegend config={blockChartConfig} />}
      >
        <LineChart accessibilityLayer data={chartData}>
          <CartesianGrid vertical={false} />
          <XAxis
            axisLine={false}
            dataKey="time"
            minTickGap={32}
            tickLine={false}
          />
          <YAxis
            axisLine={false}
            tickFormatter={(value) => formatBytes(Number(value))}
            tickLine={false}
            width={64}
          />
          <ChartTooltip
            content={
              <ChartTooltipContent
                formatter={(value) => formatBytes(Number(value))}
              />
            }
          />
          <Line
            dataKey="blockRead"
            dot={false}
            stroke="var(--color-blockRead)"
            strokeWidth={2}
            type="monotone"
          />
          <Line
            dataKey="blockWrite"
            dot={false}
            stroke="var(--color-blockWrite)"
            strokeWidth={2}
            type="monotone"
          />
        </LineChart>
      </StatsChartCard>
    </div>
  );
}
