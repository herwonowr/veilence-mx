"use client"

import { use } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { notFound } from "next/navigation"
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
import { Skeleton } from "@/components/ui/skeleton"
import { ArrowLeft, Activity, Package as PackageIcon } from "lucide-react"
import { ProtectedRoute } from "@/components/protected-route"
import { RequireOrg } from "@/components/require-org"
import { DetailError } from "@/components/detail-error"
import { usePackage, usePackageReleases, useAnalysisHistory } from "@/features/packages"
import type { Classification } from "@/types"

function classificationColor(c: Classification) {
  if (c === "malicious") return "destructive" as const
  if (c === "suspicious") return "default" as const
  return "secondary" as const
}

export default function PackageDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  return (
    <ProtectedRoute>
      <RequireOrg feature="package details">
        <PackageDetailContent params={params} />
      </RequireOrg>
    </ProtectedRoute>
  )
}

function PackageDetailContent({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  const router = useRouter()
  const packageId = parseInt(id, 10)

  // NaN validation: show 404 for non-numeric IDs
  if (isNaN(packageId) || packageId <= 0) {
    notFound()
  }

  const { data: pkgRes, isError, refetch } = usePackage(packageId)
  const { data: releasesRes } = usePackageReleases(packageId)
  const { data: historyRes } = useAnalysisHistory(packageId)

  const pkg = pkgRes?.data ?? null
  const releases = releasesRes?.data ?? []
  const analysisHistory = historyRes?.data ?? []

  if (isError) return (
    <DetailError
      message="Failed to load package details. The package may not exist or the server is unavailable."
      onRetry={() => refetch()}
      backHref="/packages"
      backLabel="Packages"
    />
  )

  if (!pkg) return (
    <div className="space-y-6 p-8">
      <Skeleton className="h-8 w-48" />
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
      </div>
      <Skeleton className="h-64" />
    </div>
  )

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/packages"
          className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Packages
        </Link>
        <h1 className="text-3xl font-bold">{pkg.name}</h1>
        <div className="flex flex-wrap items-center gap-2 mt-2">
          <Badge variant="outline">{pkg.ecosystem}</Badge>
          <span className="font-mono text-sm text-muted-foreground">v{pkg.latestVersion}</span>
          {pkg.description && (
            <span className="text-sm text-muted-foreground">— {pkg.description}</span>
          )}
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Release History ({releases.length})</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Version</TableHead>
                <TableHead>Published</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {releases.map((release) => (
                <TableRow
                  key={release.id}
                  clickable
                  onClick={() => router.push(`/releases/${release.id}`)}
                >
                  <TableCell className="font-mono font-medium">
                    <Link
                      href={`/releases/${release.id}`}
                      className="hover:underline text-primary"
                    >
                      {release.version}
                    </Link>
                  </TableCell>
                  <TableCell className="text-sm">
                    {new Date(release.publishedAt).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary">{release.status}</Badge>
                  </TableCell>
                </TableRow>
              ))}
              {releases.length === 0 && (
                <TableRow>
                  <TableCell colSpan={3} className="text-center py-8">
                    <div className="flex flex-col items-center gap-2">
                      <PackageIcon className="h-8 w-8 text-muted-foreground" />
                      <p className="text-muted-foreground">No releases tracked yet.</p>
                      <p className="text-xs text-muted-foreground">Releases will appear here once detected by the poller.</p>
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
          </div>
        </CardContent>
      </Card>

      {/* Analysis History Timeline (P2) */}
      {analysisHistory.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Activity className="h-5 w-5" />
              Analysis History
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="relative space-y-0">
              {/* Timeline line */}
              <div className="absolute left-[19px] top-3 bottom-3 w-px bg-border" aria-hidden="true" />

              {analysisHistory.map((entry, i) => (
                <div key={`${entry.releaseId}-${i}`} className="relative flex gap-4 pb-6 last:pb-0">
                  {/* Timeline dot */}
                  <div className={`relative z-10 ml-[14px] mt-1.5 flex h-[10px] w-[10px] shrink-0 items-center justify-center rounded-full ring-2 ring-background ${
                    entry.classification === "malicious" ? "bg-destructive" :
                    entry.classification === "suspicious" ? "bg-primary" :
                    "bg-muted-foreground"
                  }`}
                  />

                  <div className="flex-1 min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <Link
                        href={`/releases/${entry.releaseId}`}
                        className="font-mono text-sm font-medium hover:underline text-primary"
                      >
                        v{entry.version}
                      </Link>
                      <Badge variant={classificationColor(entry.classification)}>
                        {entry.classification}
                      </Badge>
                      <span className="text-xs text-muted-foreground">
                        {(entry.confidence * 100).toFixed(0)}% confidence
                      </span>
                    </div>
                    <p className="text-sm text-muted-foreground mt-1 line-clamp-2">
                      {entry.reasoning}
                    </p>
                    <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                      <span>{entry.analyzerType} / {entry.modelUsed}</span>
                      <span>{new Date(entry.analyzedAt).toLocaleDateString()}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
