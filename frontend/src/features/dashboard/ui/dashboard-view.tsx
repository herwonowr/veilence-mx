"use client"

import { useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { Card, CardContent, CardHeader, CardTitle } from "@/ui/components/card"
import { Badge } from "@/ui/components/badge"
import { Button } from "@/ui/components/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui/components/table"
import type { Classification } from "@/domains/common"
import { Skeleton } from "@/ui/components/skeleton"
import { TableSkeleton, type SkeletonColumn } from "@/ui/feedback/table-skeleton"
import { TableError } from "@/ui/feedback/table-error"
import { Package, Activity, AlertTriangle, Shield, Clock, CheckCircle, RefreshCw, Building2, Plus, BookOpen, CircleCheck, Circle } from "lucide-react"
import { Input } from "@/ui/components/input"
import { Label } from "@/ui/components/label"
import { Switch } from "@/ui/components/switch"
import { DashboardCharts } from "@/features/dashboard/ui/dashboard-charts"
import { TableEmptyState } from "@/ui/feedback/empty-state"
import { formatEcosystem } from "@/domains/common"
import { useAuth } from "@/core/providers/auth-provider"
import { useDashboardStats, useRecentReleases, useChartData, useDashboardStalePackages } from "@/features/dashboard/hooks/use-dashboard"

const classificationVariant = (c?: Classification) => {
  if (c === "malicious") return "destructive" as const
  if (c === "suspicious") return "default" as const
  if (c === "baseline") return "outline" as const
  return "secondary" as const
}

// ─── Onboarding State (no org selected) ─────────────────────

interface OnboardingProps {
  hasAnyOrg: boolean
  user: { firstName?: string; emailVerified?: boolean } | null
}

const DashboardOnboarding = ({ hasAnyOrg, user }: OnboardingProps) => {
  const router = useRouter()

  const steps = [
    { label: "Create your account", done: true },
    { label: "Create an organization", done: hasAnyOrg },
    { label: "Add packages to monitor", done: false },
    { label: "Review your first analysis", done: false },
  ]

  return (
    <div className="space-y-6">
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <Building2 className="h-12 w-12 text-muted-foreground mb-4" />
        <h1 className="text-2xl font-bold mb-2">
          Welcome to Veilence-MX{user?.firstName ? `, ${user.firstName}` : ""}!
        </h1>
        <p className="text-muted-foreground mb-6 max-w-md">
          {hasAnyOrg
            ? "Select an organization from the sidebar to view your supply chain monitoring dashboard."
            : "Get started by creating your first organization to begin monitoring your software supply chain."}
        </p>
        <div className="flex flex-wrap items-center gap-3">
          {!hasAnyOrg && (
            <Button onClick={() => router.push("/organizations")}>
              <Plus className="mr-2 h-4 w-4" />
              Create Organization
            </Button>
          )}
          <Button variant="outline" onClick={() => router.push("/packages")}>
            <Package className="mr-2 h-4 w-4" />
            {hasAnyOrg ? "Go to Packages" : "Browse Packages"}
          </Button>
        </div>
      </div>

      <Card className="max-w-md mx-auto">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <BookOpen className="h-4 w-4" />
            Getting Started
          </CardTitle>
        </CardHeader>
        <CardContent>
          <ol className="space-y-3">
            {steps.map((step, i) => (
              <li key={i} className="flex items-center gap-3">
                {step.done ? (
                  <CircleCheck className="h-5 w-5 text-green-500 shrink-0" />
                ) : (
                  <Circle className="h-5 w-5 text-muted-foreground shrink-0" />
                )}
                <span className={step.done ? "text-muted-foreground line-through" : "text-sm font-medium"}>
                  {step.label}
                </span>
              </li>
            ))}
          </ol>
        </CardContent>
      </Card>
    </div>
  )
}

// ─── Dashboard Data (org selected) ──────────────────────────

const DashboardData = () => {
  const router = useRouter()
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [intervalSec, setIntervalSec] = useState(30)
  const [chartRange, setChartRange] = useState<{ from?: string; to?: string }>({})

  const refetchInterval = autoRefresh && intervalSec > 0 ? intervalSec * 1000 : false

  const { data: statsRes } = useDashboardStats({
    refetchInterval,
  })
  const stats = statsRes?.data ?? null

  const { data: releasesRes, isLoading: releasesLoading, isError: releasesError, refetch: refetchReleases } = useRecentReleases(
    { page: 1, limit: 10, latestPerPackage: true },
    { refetchInterval }
  )
  const releases = releasesRes?.data ?? []

  const { data: staleRes } = useDashboardStalePackages(6, { refetchInterval })
  const stalePackages = staleRes?.data ?? []

  const releasesSkeletonColumns: SkeletonColumn[] = [
    { width: "w-28", header: "Package" },
    { width: "w-16", header: "Ecosystem" },
    { width: "w-20", header: "Version" },
    { width: "w-16", header: "Status" },
    { width: "w-20", header: "Classification" },
  ]

  const chartParams = chartRange.from || chartRange.to ? chartRange : undefined
  const { data: chartRes, isLoading: chartsLoading } = useChartData(chartParams, {
    refetchInterval,
  })
  const chartData = chartRes?.data ?? null

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <div className="flex flex-wrap items-center gap-4">
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
                aria-label="Auto-refresh interval in seconds"
              />
              <span className="text-sm text-muted-foreground">sec</span>
            </div>
          )}
          {autoRefresh && (
            <RefreshCw className="h-3.5 w-3.5 text-muted-foreground animate-spin" style={{ animationDuration: "3s" }} aria-hidden="true" />
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
            <AlertTriangle className="h-4 w-4 text-amber-500" />
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

      {!chartsLoading && chartData ? (
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
          <div className="overflow-x-auto">
          {releasesLoading ? (
            <TableSkeleton columns={releasesSkeletonColumns} rows={5} />
          ) : releasesError ? (
            <TableError colSpan={5} onRetry={() => refetchReleases()} />
          ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Package</TableHead>
                <TableHead className="hidden md:table-cell">Ecosystem</TableHead>
                <TableHead>Version</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="hidden lg:table-cell">Classification</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {releases.map((release) => (
                <TableRow
                  key={release.id}
                  clickable
                  onClick={() => router.push(`/releases/${release.id}`)}
                >
                  <TableCell className="font-medium">
                    <Link
                      href={`/releases/${release.id}`}
                      className="hover:underline text-primary"
                    >
                      {release.packageName}
                    </Link>
                  </TableCell>
                  <TableCell className="hidden md:table-cell">
                    <Badge variant="outline">{formatEcosystem(release.packageEcosystem)}</Badge>
                  </TableCell>
                  <TableCell className="font-mono text-sm">{release.version}</TableCell>
                  <TableCell>
                    {release.status === "completed" ? (
                      <span className="flex items-center gap-1 text-sm text-green-600">
                        <CheckCircle className="h-3.5 w-3.5" aria-hidden="true" />
                        Done
                      </span>
                    ) : (
                      <Badge variant="secondary">{release.status}</Badge>
                    )}
                  </TableCell>
                  <TableCell className="hidden lg:table-cell">
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
                <TableEmptyState
                  colSpan={5}
                  icon={<Activity className="h-8 w-8" />}
                  title="No releases yet."
                  description="Add packages to start monitoring releases."
                >
                  <Link href="/packages">
                    <Button variant="outline" size="sm">
                      Go to Packages
                    </Button>
                  </Link>
                </TableEmptyState>
              )}
            </TableBody>
          </Table>
          )}
          </div>
        </CardContent>
      </Card>

      {stalePackages.length > 0 && (
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-muted-foreground" />
              Stale Packages ({stalePackages.length})
            </CardTitle>
            <Link href="/packages/stale">
              <Button variant="outline" size="sm">View All</Button>
            </Link>
          </CardHeader>
          <CardContent>
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Package</TableHead>
                    <TableHead className="hidden md:table-cell">Ecosystem</TableHead>
                    <TableHead>Last Release</TableHead>
                    <TableHead>Days Since</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {stalePackages.slice(0, 5).map((pkg) => (
                    <TableRow key={pkg.id}>
                      <TableCell className="font-medium">
                        <Link
                          href={`/packages/${pkg.id}`}
                          className="hover:underline text-primary"
                        >
                          {pkg.name}
                        </Link>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <Badge variant="outline">{formatEcosystem(pkg.ecosystem)}</Badge>
                      </TableCell>
                      <TableCell className="text-sm">
                        {pkg.lastReleaseAt
                          ? new Date(pkg.lastReleaseAt).toLocaleDateString()
                          : "Never"}
                      </TableCell>
                      <TableCell>
                        <span className={pkg.daysSinceLastRelease > 365 ? "text-destructive font-medium" : ""}>
                          {pkg.daysSinceLastRelease}
                        </span>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

// ─── Main Dashboard View ────────────────────────────────────

export const DashboardView = () => {
  const { currentOrg, organizations, user } = useAuth()
  const hasOrg = !!currentOrg

  if (!hasOrg) {
    return <DashboardOnboarding hasAnyOrg={organizations.length > 0} user={user} />
  }

  return <DashboardData />
}
