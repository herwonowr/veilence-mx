"use client"

import { useState } from "react"
import { Area, AreaChart, Bar, BarChart, CartesianGrid, Cell, Label, Pie, PieChart, XAxis, YAxis } from "recharts"
import { Card, CardContent, CardHeader, CardTitle } from "@/ui/components/card"
import { EmptyState } from "@/ui/feedback/empty-state"
import { BarChart3, PieChart as PieChartIcon, TrendingUp } from "lucide-react"
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  ChartLegend,
  ChartLegendContent,
} from "@/ui/components/chart"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui/components/select"
import { Calendar } from "@/ui/components/calendar"
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/components/popover"
import { Button, buttonVariants } from "@/ui/components/button"
import { RotateCcw, CalendarIcon } from "lucide-react"
import { cn } from "@/core/utils"
import type { ChartData } from "@/domains/dashboard"

const classificationConfig = {
  benign: { label: "Benign", color: "#22c55e" },
  suspicious: { label: "Suspicious", color: "#f59e0b" },
  malicious: { label: "Malicious", color: "#ef4444" },
  baseline: { label: "Baseline", color: "#6b7280" },
} satisfies ChartConfig

const ecosystemConfig = {
  python: { label: "Python", color: "#3b82f6" },
  npm: { label: "NPM", color: "#ef4444" },
} satisfies ChartConfig

const severityConfig = {
  critical: { label: "Critical", color: "#dc2626" },
  high: { label: "High", color: "#f97316" },
  medium: { label: "Medium", color: "#eab308" },
  low: { label: "Low", color: "#6b7280" },
} satisfies ChartConfig

const statusConfig = {
  pending: { label: "Pending", color: "#f59e0b" },
  diffing: { label: "Diffing", color: "#3b82f6" },
  analyzing: { label: "Analyzing", color: "#8b5cf6" },
  completed: { label: "Completed", color: "#22c55e" },
  error: { label: "Error", color: "#ef4444" },
} satisfies ChartConfig

const activityConfig = {
  releases: { label: "Releases", color: "#3b82f6" },
} satisfies ChartConfig

interface DashboardChartsProps {
  data: ChartData
  range: { from?: string; to?: string }
  onRangeChange: (range: { from?: string; to?: string }) => void
}

const formatDate = (d: Date): string => {
  return d.toISOString().split("T")[0]
}

const getPresetRange = (preset: string): { from: string; to: string } => {
  const to = formatDate(new Date())
  const now = new Date()
  switch (preset) {
    case "24h": return { from: formatDate(new Date(now.getTime() - 86400000)), to }
    case "7d": return { from: formatDate(new Date(now.getTime() - 7 * 86400000)), to }
    case "14d": return { from: formatDate(new Date(now.getTime() - 14 * 86400000)), to }
    case "30d": return { from: formatDate(new Date(now.getTime() - 30 * 86400000)), to }
    case "90d": return { from: formatDate(new Date(now.getTime() - 90 * 86400000)), to }
    case "6m": { const d = new Date(now); d.setMonth(d.getMonth() - 6); return { from: formatDate(d), to } }
    case "1y": { const d = new Date(now); d.setFullYear(d.getFullYear() - 1); return { from: formatDate(d), to } }
    default: return { from: formatDate(new Date(now.getTime() - 30 * 86400000)), to }
  }
}

export const DashboardCharts = ({ data, range, onRangeChange }: DashboardChartsProps) => {
  const [preset, setPreset] = useState("30d")
  const [fromDate, setFromDate] = useState<Date | undefined>()
  const [toDate, setToDate] = useState<Date | undefined>()
  const [fromOpen, setFromOpen] = useState(false)
  const [toOpen, setToOpen] = useState(false)
  const isCustom = preset === "custom"

  const handlePresetChange = (value: string) => {
    setPreset(value)
    if (value === "custom") {
      return
    }
    setFromDate(undefined)
    setToDate(undefined)
    onRangeChange(getPresetRange(value))
  }

  const handleFromSelect = (date: Date | undefined) => {
    setFromDate(date)
    setFromOpen(false)
    if (date || toDate) {
      onRangeChange({
        from: date ? formatDate(date) : undefined,
        to: toDate ? formatDate(toDate) : undefined,
      })
    }
  }

  const handleToSelect = (date: Date | undefined) => {
    setToDate(date)
    setToOpen(false)
    if (fromDate || date) {
      onRangeChange({
        from: fromDate ? formatDate(fromDate) : undefined,
        to: date ? formatDate(date) : undefined,
      })
    }
  }

  const handleReset = () => {
    setPreset("30d")
    setFromDate(undefined)
    setToDate(undefined)
    onRangeChange({})
  }
  const classificationData = (data.classifications ?? []).map((c) => ({
    ...c,
    fill: classificationConfig[c.classification as keyof typeof classificationConfig]?.color ?? "#6b7280",
  }))

  const ecosystemData = (data.ecosystems ?? []).map((r) => ({
    ...r,
    fill: ecosystemConfig[r.ecosystem as keyof typeof ecosystemConfig]?.color ?? "#6b7280",
  }))

  const statusData = (data.releaseStatuses ?? []).map((s) => ({
    ...s,
    fill: statusConfig[s.status as keyof typeof statusConfig]?.color ?? "#6b7280",
  }))

  const totalClassifications = classificationData.reduce((sum, c) => sum + c.count, 0)
  const totalPackages = ecosystemData.reduce((sum, r) => sum + r.count, 0)

  const presetLabels: Record<string, string> = {
    "24h": "Last 24 Hours",
    "7d": "Last 7 Days",
    "14d": "Last 14 Days",
    "30d": "Last 30 Days",
    "90d": "Last 90 Days",
    "6m": "Last 6 Months",
    "1y": "Last Year",
    custom: "Custom Range",
  }

  const activityTitle = isCustom && (range.from || range.to)
    ? `Release Activity (${range.from ?? "..."} - ${range.to ?? "..."})`
    : `Release Activity (${presetLabels[preset] ?? "Last 30 Days"})`

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <Select value={preset} onValueChange={(v) => { if (v) handlePresetChange(v) }}>
          <SelectTrigger className="w-44" aria-label="Chart time range">
            <SelectValue>{presetLabels[preset] ?? preset}</SelectValue>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="24h">Last 24 Hours</SelectItem>
            <SelectItem value="7d">Last 7 Days</SelectItem>
            <SelectItem value="14d">Last 14 Days</SelectItem>
            <SelectItem value="30d">Last 30 Days</SelectItem>
            <SelectItem value="90d">Last 90 Days</SelectItem>
            <SelectItem value="6m">Last 6 Months</SelectItem>
            <SelectItem value="1y">Last Year</SelectItem>
            <SelectItem value="custom">Custom Range</SelectItem>
          </SelectContent>
        </Select>
        {/* Isolate date pickers from Select's event scope - without this,
            calendar click events bubble through React's synthetic tree and
            get intercepted by the Base UI Select, resetting the preset.
            The wrapper stops propagation for the triggers; each PopoverContent
            also stops propagation because it renders through a portal (outside
            the wrapper in the DOM but still a React child of the Select). */}
        {isCustom && (
          <div
            className="contents"
            onClick={(e) => e.stopPropagation()}
            onPointerDown={(e) => e.stopPropagation()}
          >
            <Popover open={fromOpen} onOpenChange={setFromOpen}>
              <PopoverTrigger
                render={
                  <button
                    type="button"
                    className={cn(
                      buttonVariants({ variant: "outline", size: "sm" }),
                      "w-40 justify-start text-left font-normal"
                    )}
                    aria-label="Select start date"
                  />
                }
              >
                <CalendarIcon className="mr-2 h-4 w-4" />
                {fromDate ? fromDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }) : "Start date"}
              </PopoverTrigger>
              <PopoverContent
                align="start"
                className="w-auto"
                onClick={(e: React.MouseEvent) => e.stopPropagation()}
                onPointerDown={(e: React.PointerEvent) => e.stopPropagation()}
              >
                <Calendar
                  mode="single"
                  selected={fromDate}
                  onSelect={handleFromSelect}
                  disabled={(date) => toDate ? date > toDate : date > new Date()}
                  defaultMonth={fromDate}
                />
              </PopoverContent>
            </Popover>
            <span className="text-sm text-muted-foreground">to</span>
            <Popover open={toOpen} onOpenChange={setToOpen}>
              <PopoverTrigger
                render={
                  <button
                    type="button"
                    className={cn(
                      buttonVariants({ variant: "outline", size: "sm" }),
                      "w-40 justify-start text-left font-normal"
                    )}
                    aria-label="Select end date"
                  />
                }
              >
                <CalendarIcon className="mr-2 h-4 w-4" />
                {toDate ? toDate.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }) : "End date"}
              </PopoverTrigger>
              <PopoverContent
                align="start"
                className="w-auto"
                onClick={(e: React.MouseEvent) => e.stopPropagation()}
                onPointerDown={(e: React.PointerEvent) => e.stopPropagation()}
              >
                <Calendar
                  mode="single"
                  selected={toDate}
                  onSelect={handleToSelect}
                  disabled={(date) => fromDate ? date < fromDate || date > new Date() : date > new Date()}
                  defaultMonth={toDate ?? fromDate}
                />
              </PopoverContent>
            </Popover>
          </div>
        )}
        {(range.from || range.to) && (
          <Button variant="ghost" size="sm" onClick={handleReset}>
            <RotateCcw className="mr-1 h-3 w-3" />
            Reset
          </Button>
        )}
      </div>

    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
      {/* Release Activity - Area Chart */}
      <Card className="lg:col-span-2">
        <CardHeader className="pb-2">
          <CardTitle className="text-base">{activityTitle}</CardTitle>
        </CardHeader>
        <CardContent>
          {(data.releaseActivity ?? []).length === 0 ? (
            <EmptyState
              icon={<TrendingUp className="h-8 w-8" />}
              title="No data for this period"
              description="Release activity will appear here once packages are monitored."
              className="h-62.5"
            />
          ) : (
          <ChartContainer config={activityConfig} className="h-62.5 w-full">
            <AreaChart accessibilityLayer data={data.releaseActivity ?? []}>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="date"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                tickFormatter={(v: string) => {
                  const d = new Date(v)
                  return d.toLocaleDateString("en-US", { month: "short", day: "numeric" })
                }}
                interval="preserveStartEnd"
                minTickGap={40}
              />
              <YAxis
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                allowDecimals={false}
              />
              <ChartTooltip
                content={
                  <ChartTooltipContent
                    labelFormatter={(v) => {
                      return new Date(String(v)).toLocaleDateString("en-US", {
                        month: "long",
                        day: "numeric",
                        year: "numeric",
                      })
                    }}
                  />
                }
              />
              <Area
                dataKey="releases"
                type="monotone"
                fill="var(--color-releases)"
                fillOpacity={0.2}
                stroke="var(--color-releases)"
                strokeWidth={2}
              />
            </AreaChart>
          </ChartContainer>
          )}
        </CardContent>
      </Card>

      {/* Classification Distribution - Pie Chart */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">Classification Distribution</CardTitle>
        </CardHeader>
        <CardContent>
          {totalClassifications === 0 ? (
            <EmptyState
              icon={<PieChartIcon className="h-8 w-8" />}
              title="No data for this period"
              description="Classification data will appear after releases are analyzed."
              className="aspect-square h-62.5 mx-auto"
            />
          ) : (
          <ChartContainer config={classificationConfig} className="mx-auto aspect-square h-62.5">
            <PieChart>
              <ChartTooltip content={<ChartTooltipContent nameKey="classification" hideLabel />} />
              <Pie
                data={classificationData}
                dataKey="count"
                nameKey="classification"
                innerRadius={60}
                strokeWidth={4}
              >
                <Label
                  content={({ viewBox }) => {
                    if (viewBox && "cx" in viewBox && "cy" in viewBox) {
                      return (
                        <text x={viewBox.cx} y={viewBox.cy} textAnchor="middle" dominantBaseline="middle">
                          <tspan x={viewBox.cx} y={viewBox.cy} className="fill-foreground text-2xl font-bold">
                            {totalClassifications}
                          </tspan>
                          <tspan x={viewBox.cx} y={(viewBox.cy ?? 0) + 20} className="fill-muted-foreground text-xs">
                            Analyzed
                          </tspan>
                        </text>
                      )
                    }
                    return null
                  }}
                />
              </Pie>
              <ChartLegend content={<ChartLegendContent nameKey="classification" />} />
            </PieChart>
          </ChartContainer>
          )}
        </CardContent>
      </Card>

      {/* Ecosystem Distribution - Pie Chart */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">Packages by Ecosystem</CardTitle>
        </CardHeader>
        <CardContent>
          {totalPackages === 0 ? (
            <EmptyState
              icon={<PieChartIcon className="h-8 w-8" />}
              title="No data for this period"
              description="Package ecosystem data will appear once packages are added."
              className="aspect-square h-62.5 mx-auto"
            />
          ) : (
          <ChartContainer config={ecosystemConfig} className="mx-auto aspect-square h-62.5">
            <PieChart>
              <ChartTooltip content={<ChartTooltipContent nameKey="ecosystem" hideLabel />} />
              <Pie
                data={ecosystemData}
                dataKey="count"
                nameKey="ecosystem"
                innerRadius={60}
                strokeWidth={4}
              >
                <Label
                  content={({ viewBox }) => {
                    if (viewBox && "cx" in viewBox && "cy" in viewBox) {
                      return (
                        <text x={viewBox.cx} y={viewBox.cy} textAnchor="middle" dominantBaseline="middle">
                          <tspan x={viewBox.cx} y={viewBox.cy} className="fill-foreground text-2xl font-bold">
                            {totalPackages}
                          </tspan>
                          <tspan x={viewBox.cx} y={(viewBox.cy ?? 0) + 20} className="fill-muted-foreground text-xs">
                            Packages
                          </tspan>
                        </text>
                      )
                    }
                    return null
                  }}
                />
              </Pie>
              <ChartLegend content={<ChartLegendContent nameKey="ecosystem" />} />
            </PieChart>
          </ChartContainer>
          )}
        </CardContent>
      </Card>

      {/* Alert Severity - Bar Chart */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">Alerts by Severity</CardTitle>
        </CardHeader>
        <CardContent>
          {data.alertsBySeverity.reduce((sum, s) => sum + s.count, 0) === 0 ? (
            <EmptyState
              icon={<BarChart3 className="h-8 w-8" />}
              title="No data for this period"
              description="Alert severity data will appear when alerts are generated."
              className="h-62.5"
            />
          ) : (
          <ChartContainer config={severityConfig} className="h-62.5 w-full">
            <BarChart accessibilityLayer data={data.alertsBySeverity}>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="severity"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
                tickFormatter={(v: string) => v.charAt(0).toUpperCase() + v.slice(1)}
              />
              <YAxis tickLine={false} axisLine={false} tickMargin={8} allowDecimals={false} />
              <ChartTooltip content={<ChartTooltipContent hideLabel />} />
              <Bar dataKey="count" radius={[4, 4, 0, 0]}>
                {data.alertsBySeverity.map((entry) => (
                  <Cell
                    key={entry.severity}
                    fill={severityConfig[entry.severity as keyof typeof severityConfig]?.color ?? "#6b7280"}
                  />
                ))}
              </Bar>
            </BarChart>
          </ChartContainer>
          )}
        </CardContent>
      </Card>

      {/* Release Status - Pie Chart */}
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-base">Release Processing Status</CardTitle>
        </CardHeader>
        <CardContent>
          {statusData.reduce((sum, s) => sum + s.count, 0) === 0 ? (
            <EmptyState
              icon={<PieChartIcon className="h-8 w-8" />}
              title="No data for this period"
              description="Release status data will appear once releases are processed."
              className="aspect-square h-62.5 mx-auto"
            />
          ) : (
          <ChartContainer config={statusConfig} className="mx-auto aspect-square h-62.5">
            <PieChart>
              <ChartTooltip content={<ChartTooltipContent nameKey="status" hideLabel />} />
              <Pie
                data={statusData}
                dataKey="count"
                nameKey="status"
                innerRadius={60}
                strokeWidth={4}
              >
                <Label
                  content={({ viewBox }) => {
                    if (viewBox && "cx" in viewBox && "cy" in viewBox) {
                      const total = statusData.reduce((sum, s) => sum + s.count, 0)
                      return (
                        <text x={viewBox.cx} y={viewBox.cy} textAnchor="middle" dominantBaseline="middle">
                          <tspan x={viewBox.cx} y={viewBox.cy} className="fill-foreground text-2xl font-bold">
                            {total}
                          </tspan>
                          <tspan x={viewBox.cx} y={(viewBox.cy ?? 0) + 20} className="fill-muted-foreground text-xs">
                            Releases
                          </tspan>
                        </text>
                      )
                    }
                    return null
                  }}
                />
              </Pie>
              <ChartLegend content={<ChartLegendContent nameKey="status" />} />
            </PieChart>
          </ChartContainer>
          )}
        </CardContent>
      </Card>
    </div>
    </div>
  )
}
