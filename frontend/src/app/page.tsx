"use client"

import { useState } from "react"
import Link from "next/link"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { Classification } from "@/types"
import { Skeleton } from "@/components/ui/skeleton"
import { Package, Activity, AlertTriangle, Shield, Clock, CheckCircle, RefreshCw } from "lucide-react"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { DashboardCharts } from "@/components/dashboard-charts"
import { ProtectedRoute } from "@/components/protected-route"
import { useDashboardStats, useRecentReleases, useChartData } from "@/features/dashboard"

function classificationVariant(c?: Classification) {
  if (c === "malicious") return "destructive" as const
  if (c === "suspicious") return "default" as const
  if (c === "baseline") return "outline" as const
  return "secondary" as const
}

export default function DashboardPage() {
  return (
    <ProtectedRoute>
      <DashboardContent />
    </ProtectedRoute>
  )
}

function DashboardContent() {
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [intervalSec, setIntervalSec] = useState(30)
  const [chartRange, setChartRange] = useState<{ from?: string; to?: string }>({})

  const refetchInterval = autoRefresh && intervalSec > 0 ? intervalSec * 1000 : false

  const { data: statsRes } = useDashboardStats({
    refetchInterval,
  })
  const stats = statsRes?.data ?? null

  const { data: releasesRes } = useRecentReleases(
    { page: 1, limit: 15, latestPerPackage: true },
    { refetchInterval }
  )
  const releases = releasesRes?.data ?? []

  const chartParams = chartRange.from || chartRange.to ? chartRange : undefined
  const { data: chartRes } = useChartData(chartParams, {
    refetchInterval,
  })
  const chartData = chartRes?.data ?? null

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <Switch
              id="auto-refresh"
              checked={autoRefresh}
              onCheckedChange={setAutoRefresh}
            />
            <Label htmlFor="auto-refresh" className="text-sm text-muted-foreground whitespace-nowrap">
              Auto-refresh
            </Label>
          </div>
          {autoRefresh && (
            <div className="flex items-center gap-1.5">
              <Input
                type="number"
                min={5}
                max={300}
                value={intervalSec}
                onChange={(e) => {
                  const v = parseInt(e.target.value, 10)
                  if (!isNaN(v) && v >= 5) setIntervalSec(v)
                }}
                className="w-16 h-8 text-sm"
              />
              <span className="text-sm text-muted-foreground">sec</span>
            </div>
          )}
          {autoRefresh && (
            <RefreshCw className="h-3.5 w-3.5 text-muted-foreground animate-spin" style={{ animationDuration: "3s" }} />
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Packages</CardTitle>
            <Package className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {stats ? <div className="text-2xl font-bold">{stats.totalPackages}</div> : <Skeleton className="h-8 w-12" />}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Releases</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {stats ? <div className="text-2xl font-bold">{stats.totalReleases}</div> : <Skeleton className="h-8 w-12" />}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Pending</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {stats ? <div className="text-2xl font-bold">{stats.pendingAnalyses}</div> : <Skeleton className="h-8 w-12" />}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Active Alerts</CardTitle>
            <AlertTriangle className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            {stats ? <div className="text-2xl font-bold">{stats.activeAlerts}</div> : <Skeleton className="h-8 w-12" />}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Malicious</CardTitle>
            <Shield className="h-4 w-4 text-destructive" />
          </CardHeader>
          <CardContent>
            {stats ? <div className="text-2xl font-bold text-destructive">{stats.recentMalicious}</div> : <Skeleton className="h-8 w-12" />}
          </CardContent>
        </Card>
      </div>

      {chartData ? (
        <DashboardCharts data={chartData} range={chartRange} onRangeChange={setChartRange} />
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Card key={i} className={i === 0 ? "lg:col-span-2" : ""}>
              <CardHeader className="pb-2">
                <Skeleton className="h-5 w-40" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-62.5 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Recent Releases</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Package</TableHead>
                <TableHead>Registry</TableHead>
                <TableHead>Version</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Classification</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {releases.map((release) => (
                <TableRow key={release.id}>
                  <TableCell className="font-medium">
                    <Link
                      href={`/releases/${release.id}`}
                      className="hover:underline text-primary"
                    >
                      {release.packageName}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{release.packageRegistry}</Badge>
                  </TableCell>
                  <TableCell className="font-mono text-sm">{release.version}</TableCell>
                  <TableCell>
                    {release.status === "completed" ? (
                      <span className="flex items-center gap-1 text-sm text-green-600">
                        <CheckCircle className="h-3.5 w-3.5" />
                        Done
                      </span>
                    ) : (
                      <Badge variant="secondary">{release.status}</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {release.classification ? (
                      <Badge variant={classificationVariant(release.classification)}>
                        {release.classification}
                      </Badge>
                    ) : (
                      <span className="text-muted-foreground text-sm">—</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
              {releases.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                    No releases yet. Add packages to start monitoring.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
