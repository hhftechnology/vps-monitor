import { useMemo } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

import type { ContainerStats } from "../types/stats";

interface ContainerStatsChartsProps {
  history: ContainerStats[];
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / k ** i).toFixed(1)} ${sizes[i]}`;
}

function formatTime(timestamp: number): string {
  const date = new Date(timestamp * 1000);
  return date.toLocaleTimeString("en-US", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

interface ChartData {
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

  if (history.length === 0) {
    return (
      <div className="py-8 text-center text-muted-foreground text-sm">
        No data yet. Stats will appear once streaming begins.
      </div>
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-2">
      {/* CPU Chart */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">CPU Usage (%)</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-[200px]">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="cpuGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis
                  dataKey="time"
                  tick={{ fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                  className="text-muted-foreground"
                />
                <YAxis
                  domain={[0, 100]}
                  tick={{ fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(value) => `${value}%`}
                  className="text-muted-foreground"
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: "var(--radius)",
                  }}
                  labelStyle={{ color: "hsl(var(--foreground))" }}
                  formatter={(value) => [`${(value as number).toFixed(2)}%`, "CPU"]}
                />
                <Area
                  type="monotone"
                  dataKey="cpu"
                  stroke="hsl(var(--primary))"
                  fill="url(#cpuGradient)"
                  strokeWidth={2}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>

      {/* Memory Chart */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Memory Usage (%)</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-[200px]">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="memoryGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="hsl(var(--chart-2))" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="hsl(var(--chart-2))" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis
                  dataKey="time"
                  tick={{ fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                  className="text-muted-foreground"
                />
                <YAxis
                  domain={[0, 100]}
                  tick={{ fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(value) => `${value}%`}
                  className="text-muted-foreground"
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: "var(--radius)",
                  }}
                  labelStyle={{ color: "hsl(var(--foreground))" }}
                  formatter={(value, _name, props) => {
                    const payload = (props as { payload: ChartData }).payload;
                    return [
                      `${(value as number).toFixed(2)}% (${formatBytes(payload.memoryUsage)} / ${formatBytes(payload.memoryLimit)})`,
                      "Memory",
                    ];
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="memory"
                  stroke="hsl(var(--chart-2))"
                  fill="url(#memoryGradient)"
                  strokeWidth={2}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>

      {/* Network I/O Chart */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Network I/O</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-[200px]">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="rxGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="hsl(var(--chart-3))" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="hsl(var(--chart-3))" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="txGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="hsl(var(--chart-4))" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="hsl(var(--chart-4))" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis
                  dataKey="time"
                  tick={{ fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                  className="text-muted-foreground"
                />
                <YAxis
                  tick={{ fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(value) => formatBytes(value)}
                  className="text-muted-foreground"
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: "var(--radius)",
                  }}
                  labelStyle={{ color: "hsl(var(--foreground))" }}
                  formatter={(value, name) => [
                    formatBytes(value as number),
                    name === "networkRx" ? "RX (Received)" : "TX (Transmitted)",
                  ]}
                />
                <Area
                  type="monotone"
                  dataKey="networkRx"
                  stroke="hsl(var(--chart-3))"
                  fill="url(#rxGradient)"
                  strokeWidth={2}
                  name="networkRx"
                />
                <Area
                  type="monotone"
                  dataKey="networkTx"
                  stroke="hsl(var(--chart-4))"
                  fill="url(#txGradient)"
                  strokeWidth={2}
                  name="networkTx"
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
          <div className="mt-2 flex justify-center gap-4 text-xs text-muted-foreground">
            <div className="flex items-center gap-1">
              <div className="size-2 rounded-full" style={{ backgroundColor: "hsl(var(--chart-3))" }} />
              <span>RX (Received)</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="size-2 rounded-full" style={{ backgroundColor: "hsl(var(--chart-4))" }} />
              <span>TX (Transmitted)</span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Block I/O Chart */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">Block I/O</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-[200px]">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="readGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="hsl(var(--chart-5))" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="hsl(var(--chart-5))" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="writeGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="hsl(var(--destructive))" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="hsl(var(--destructive))" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                <XAxis
                  dataKey="time"
                  tick={{ fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                  className="text-muted-foreground"
                />
                <YAxis
                  tick={{ fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(value) => formatBytes(value)}
                  className="text-muted-foreground"
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: "hsl(var(--card))",
                    border: "1px solid hsl(var(--border))",
                    borderRadius: "var(--radius)",
                  }}
                  labelStyle={{ color: "hsl(var(--foreground))" }}
                  formatter={(value, name) => [
                    formatBytes(value as number),
                    name === "blockRead" ? "Read" : "Write",
                  ]}
                />
                <Area
                  type="monotone"
                  dataKey="blockRead"
                  stroke="hsl(var(--chart-5))"
                  fill="url(#readGradient)"
                  strokeWidth={2}
                  name="blockRead"
                />
                <Area
                  type="monotone"
                  dataKey="blockWrite"
                  stroke="hsl(var(--destructive))"
                  fill="url(#writeGradient)"
                  strokeWidth={2}
                  name="blockWrite"
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
          <div className="mt-2 flex justify-center gap-4 text-xs text-muted-foreground">
            <div className="flex items-center gap-1">
              <div className="size-2 rounded-full" style={{ backgroundColor: "hsl(var(--chart-5))" }} />
              <span>Read</span>
            </div>
            <div className="flex items-center gap-1">
              <div className="size-2 rounded-full" style={{ backgroundColor: "hsl(var(--destructive))" }} />
              <span>Write</span>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
