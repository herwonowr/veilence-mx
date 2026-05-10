"use client"

import { useState } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useLocalStorage, useAuth, ROUTES , usePublicConfigQuery } from "@/core"
import { Card, CardContent, CardHeader, CardTitle, Badge, Button, Skeleton, TableSkeleton, TableError, Alert, AlertDescription, Label, Switch, TableEmptyState, Select, SelectTrigger, SelectValue, SelectContent, SelectItem, type SkeletonColumn } from "@/ui"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui"
import type { Classification } from "@/domains/common"
import { Package, Activity, AlertTriangle, Shield, Clock, CheckCircle, RefreshCw, Layers, Plus, Mail } from "lucide-react"
import { DashboardCharts } from "@/features/dashboard/ui/dashboard-charts"
import { formatEcosystem } from "@/domains/common"
import { formatVersion } from "@/domains/releases"
import { useQuery } from "@tanstack/react-query"
import { apiGetMyInvitations, myInvitationKeys } from "@/domains/admin"
import { useDashboardStats, useRecentReleases, useChartData, useDashboardStalePackages, useDashboardSettings } from "@/features/dashboard/hooks/use-dashboard"
import { PendingSuggestionsCard } from "@/features/dashboard/ui/pending-suggestions-card"

const classificationVariant = (c?: Classification) => {
  if (c === "malicious") return "destructive" as const
  if (c === "suspicious") return "default" as const
  if (c === "baseline") return "outline" as const
  return "secondary" as const
}

// ─── Onboarding State (no workspace selected) ─────────────────────

interface OnboardingProps {
  hasAnyWorkspace: boolean
  user: { firstName?: string; emailVerified?: boolean } | null
}

const DashboardOnboarding = ({ hasAnyWorkspace, user }: OnboardingProps) => {
  const router = useRouter()
  const { registrationEnabled } = usePublicConfigQuery()
  const { data: invitationsRes } = useQuery({
    queryKey: myInvitationKeys.all,
    queryFn: () => apiGetMyInvitations(),
    enabled: registrationEnabled,
  })
  const pendingCount = (invitationsRes?.data ?? []).filter(
    (inv) => inv.status === "pending"
  ).length

  return (
    <div className="space-y-6">
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <Layers className="h-12 w-12 text-muted-foreground mb-4" />
        <h1 className="text-2xl font-bold mb-2">
          Welcome to Veilence-MX{user?.firstName ? `, ${user.firstName}` : ""}!
        </h1>
        <p className="text-muted-foreground mb-6 max-w-md">
          {hasAnyWorkspace
            ? "Select a workspace from the sidebar to view your supply chain monitoring dashboard."
            : "Get started by creating your first workspace to begin monitoring your software supply chain."}
        </p>
        <div className="flex flex-wrap items-center gap-3">
          {!hasAnyWorkspace && (
            <Button onClick={() => router.push(ROUTES.WORKSPACES)}>
              <Plus className="mr-2 h-4 w-4" />
              Create Workspace
            </Button>
          )}
          <Button variant="outline" onClick={() => router.push(ROUTES.PACKAGES)}>
            <Package className="mr-2 h-4 w-4" />
            {hasAnyWorkspace ? "Go to Packages" : "Browse Packages"}
          </Button>
        </div>
      </div>

      {registrationEnabled && pendingCount > 0 && (
        <Card className="max-w-md mx-auto">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Mail className="h-4 w-4" />
              Pending Invitations
              <Badge variant="destructive">{pendingCount}</Badge>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground mb-3">
              You have pending workspace invitations waiting for your response.
            </p>
            <Link href={ROUTES.WORKSPACES_INVITATIONS}>
              <Button variant="outline" size="sm">
                <Mail className="mr-2 h-4 w-4" />
                View Invitations
              </Button>
            </Link>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

// ─── Dashboard Data (workspace selected) ──────────────────────────

const DashboardData = () => {
  const router = useRouter()
  const [autoRefresh, setAutoRefresh] = useLocalStorage("vmx-dashboard-auto-refresh", true)
  const [intervalSec, setIntervalSec] = useLocalStorage("vmx-dashboard-refresh-interval", 30)
  const [chartRange, setChartRange] = useState<{ from?: string; to?: string }>({})

  const refetchInterval = autoRefresh && intervalSec > 0 ? intervalSec * 1000 : false

  const { registrationEnabled } = usePublicConfigQuery()
  const { data: invitationsRes } = useQuery({
    queryKey: myInvitationKeys.all,
    queryFn: () => apiGetMyInvitations(),
    enabled: registrationEnabled,
  })
  const pendingCount = (invitationsRes?.data ?? []).filter(
    (inv) => inv.status === "pending"
  ).length

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

  const { data: settingsRes } = useDashboardSettings({ refetchInterval })
  const warningThreshold = parseInt(settingsRes?.data?.package_count_warning_threshold ?? "0", 10)

  const releasesSkeletonColumns: SkeletonColumn[] = [
    { width: "w-28", header: "Package" },
    { width: "w-16", header: "Ecosystem" },
    { width: "w-20", header: "Version" },
    { width: "w-16", header: "Analysis Status" },
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
          {registrationEnabled && pendingCount > 0 && (
            <Link href={ROUTES.WORKSPACES_INVITATIONS} className="flex items-center gap-2">
              <Button variant="outline" size="sm">
                <Mail className="mr-2 h-4 w-4" />
                Pending Invitations
                <Badge variant="destructive" className="ml-2">
                  {pendingCount}
                </Badge>
              </Button>
            </Link>
          )}
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
            <Select
              value={String(intervalSec)}
              onValueChange={(val) => setIntervalSec(Number(val))}
            >
              <SelectTrigger className="h-8 w-auto text-sm" aria-label="Auto-refresh interval">
                <SelectValue>
                  {intervalSec === 30 ? "30 Seconds" : intervalSec === 60 ? "1 Minute" : intervalSec === 300 ? "5 Minutes" : "10 Minutes"}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="30">30 Seconds</SelectItem>
                <SelectItem value="60">1 Minute</SelectItem>
                <SelectItem value="300">5 Minutes</SelectItem>
                <SelectItem value="600">10 Minutes</SelectItem>
              </SelectContent>
            </Select>
          )}
          {autoRefresh && (
            <RefreshCw className="h-3.5 w-3.5 text-muted-foreground animate-spin" style={{ animationDuration: "3s" }} aria-hidden="true" />
          )}
        </div>
      </div>

      <PendingSuggestionsCard />

      {warningThreshold > 0 && stats && stats.totalPackages > warningThreshold && (
        <Alert variant="warning">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            You are monitoring <strong>{stats.totalPackages}</strong> packages, which exceeds your warning threshold of <strong>{warningThreshold}</strong>.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
        <Link href={ROUTES.PACKAGES} className="group/stat-link">
          <Card className="cursor-pointer transition-all hover:border-l-2! group-hover/stat-link:border-l-2! h-full">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Packages</CardTitle>
              <Package className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              {stats ? <div className="text-2xl font-bold">{stats.totalPackages}</div> : <Skeleton className="h-8 w-12" />}
            </CardContent>
          </Card>
        </Link>

        <Link href={ROUTES.RELEASES} className="group/stat-link">
          <Card className="cursor-pointer transition-all hover:border-l-2! group-hover/stat-link:border-l-2! h-full">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Releases</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              {stats ? <div className="text-2xl font-bold">{stats.totalReleases}</div> : <Skeleton className="h-8 w-12" />}
            </CardContent>
          </Card>
        </Link>

        <Link href={`${ROUTES.RELEASES}?status=in_progress`} className="group/stat-link">
          <Card className="cursor-pointer transition-all hover:border-l-2! group-hover/stat-link:border-l-2! h-full">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Pending</CardTitle>
              <Clock className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              {stats ? <div className="text-2xl font-bold">{stats.pendingAnalyses}</div> : <Skeleton className="h-8 w-12" />}
            </CardContent>
          </Card>
        </Link>

        <Link href={`${ROUTES.ALERTS}?status=new`} className="group/stat-link">
          <Card className="cursor-pointer transition-all hover:border-l-2! group-hover/stat-link:border-l-2! h-full">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Active Alerts</CardTitle>
              <AlertTriangle className="h-4 w-4 text-amber-500" />
            </CardHeader>
            <CardContent>
              {stats ? <div className="text-2xl font-bold">{stats.activeAlerts}</div> : <Skeleton className="h-8 w-12" />}
            </CardContent>
          </Card>
        </Link>

        <Link href={`${ROUTES.RELEASES}?classification=malicious`} className="group/stat-link">
          <Card className="cursor-pointer transition-all hover:border-l-2! group-hover/stat-link:border-l-2! h-full">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium">Malicious</CardTitle>
              <Shield className="h-4 w-4 text-destructive" />
            </CardHeader>
            <CardContent>
              {stats ? <div className="text-2xl font-bold text-destructive">{stats.recentMalicious}</div> : <Skeleton className="h-8 w-12" />}
            </CardContent>
          </Card>
        </Link>
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
                <TableHead>Analysis Status</TableHead>
                <TableHead className="hidden lg:table-cell">Classification</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {releases.map((release) => (
                <TableRow
                  key={release.id}
                  clickable
                  onClick={() => router.push(ROUTES.RELEASE_DETAIL(release.id))}
                >
                  <TableCell className="font-medium">
                    <Link
                      href={ROUTES.RELEASE_DETAIL(release.id)}
                      className="hover:underline text-primary"
                    >
                      {release.packageName}
                    </Link>
                  </TableCell>
                  <TableCell className="hidden md:table-cell">
                    <Badge variant="outline">{formatEcosystem(release.packageEcosystem)}</Badge>
                  </TableCell>
                  <TableCell className="font-mono text-sm">{formatVersion(release.version)}</TableCell>
                  <TableCell>
                    {release.status === "completed" ? (
                      <span className="flex items-center gap-1 text-sm text-green-600">
                        <CheckCircle className="h-3.5 w-3.5" aria-hidden="true" />
                        Completed
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
                      <span className="text-muted-foreground text-sm">-</span>
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
                  <Link href={ROUTES.PACKAGES}>
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

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="flex items-center gap-2">
            <Clock className="h-5 w-5 text-muted-foreground" />
            Stale Packages ({stalePackages.length})
          </CardTitle>
          <Link href={ROUTES.PACKAGES_STALE}>
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
                  <TableHead className="hidden lg:table-cell">Days Since</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {stalePackages.slice(0, 5).map((pkg) => (
                  <TableRow
                    key={pkg.id}
                    clickable
                    onClick={() => router.push(ROUTES.PACKAGE_DETAIL(pkg.id))}
                  >
                    <TableCell className="font-medium">
                      <Link
                        href={ROUTES.PACKAGE_DETAIL(pkg.id)}
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
                    <TableCell className="hidden lg:table-cell">
                      <span className={pkg.daysSinceLastRelease > 365 ? "text-destructive font-medium" : ""}>
                        {pkg.daysSinceLastRelease}
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
                {stalePackages.length === 0 && (
                  <TableEmptyState
                    colSpan={4}
                    icon={<Clock className="h-8 w-8" />}
                    title="No stale packages."
                    description="All monitored packages have had recent releases."
                  >
                    <Link href={ROUTES.PACKAGES}>
                      <Button variant="outline" size="sm">
                        Go to Packages
                      </Button>
                    </Link>
                  </TableEmptyState>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

// ─── Main Dashboard View ────────────────────────────────────

export const DashboardView = () => {
  const { currentWorkspace, workspaces, user } = useAuth()
  const hasWorkspace = !!currentWorkspace

  if (!hasWorkspace) {
    return <DashboardOnboarding hasAnyWorkspace={workspaces.length > 0} user={user} />
  }

  return <DashboardData />
}
