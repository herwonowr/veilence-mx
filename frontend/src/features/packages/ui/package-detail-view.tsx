"use client"

import { use } from "react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { notFound } from "next/navigation"
import { Card, CardContent, CardHeader, CardTitle, Badge, Skeleton, DetailError } from "@/ui"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui"
import { ArrowLeft, Activity, Package as PackageIcon, Ban, Info } from "lucide-react"
import { formatEcosystem } from "@/domains/common"
import { formatPopularity, formatFreshness } from "@/domains/packages"
import { usePackage, usePackageReleases, useAnalysisHistory } from "@/features/packages/hooks/use-packages"
import type { Classification } from "@/domains/common"

const classificationColor = (c: Classification) => {
  if (c === "malicious") return "destructive" as const
  if (c === "suspicious") return "default" as const
  return "secondary" as const
}

export const PackageDetailView = ({
  params,
}: {
  params: Promise<{ id: string }>
}) => {
  const { id } = use(params)
  const router = useRouter()
  const packageId = id

  // Validation: show 404 for empty IDs
  if (!packageId) {
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
          className="w-fit text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Packages
        </Link>
        <h1 className="text-3xl font-bold">{pkg.name}</h1>
        <div className="flex flex-wrap items-center gap-2 mt-2">
          <Badge variant="outline">{formatEcosystem(pkg.ecosystem)}</Badge>
          <Badge variant={pkg.source === "manual" ? "default" : pkg.source === "imported" ? "outline" : "secondary"}>
            {pkg.source === "manual" ? "Manual" : pkg.source === "imported" ? "Imported" : "Discovered"}
          </Badge>
          {pkg.status === "blocked" && (
            <Badge variant="destructive">
              <Ban className="h-3 w-3 mr-1" />
              Blocked
            </Badge>
          )}
          {pkg.status === "suggested" && (
            <Badge variant="outline">Suggested</Badge>
          )}
          <span className="font-mono text-sm text-muted-foreground">v{pkg.latestVersion}</span>
          <span className="text-sm text-muted-foreground">
            {formatPopularity(pkg.ecosystem, pkg.downloadCount)}
          </span>
          {pkg.downloadCountUpdatedAt && (
            <span className="text-xs text-muted-foreground">
              ({formatFreshness(pkg.downloadCountUpdatedAt)})
            </span>
          )}
          {pkg.description && (
            <span className="text-sm text-muted-foreground">{"-"} {pkg.description}</span>
          )}
        </div>
        {pkg.status === "blocked" && pkg.blockedReason && (
          <p className="text-sm text-muted-foreground mt-2">
            Blocked: {pkg.blockedReason}
          </p>
        )}
      </div>

      {pkg.status === "suggested" && analysisHistory.length > 0 && (
        <div className="flex gap-3 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-800 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-300">
          <Info className="mt-0.5 h-4 w-4 shrink-0" />
          <div>
            <p className="font-medium">Previously monitored</p>
            <p className="mt-0.5 text-blue-700 dark:text-blue-400">
              This package was previously monitored and has historical analysis data.
              Releases during the monitoring gap may not have been analyzed.
            </p>
          </div>
        </div>
      )}

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
                <TableHead>Analysis Status</TableHead>
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
              <div className="absolute left-4.75 top-3 bottom-3 w-px bg-border" aria-hidden="true" />

              {analysisHistory.map((entry, i) => (
                <div key={`${entry.releaseId}-${i}`} className="relative flex gap-4 pb-6 last:pb-0">
                  {/* Timeline dot */}
                  <div className={`relative z-10 ml-3.5 mt-1.5 flex h-2.5 w-2.5 shrink-0 items-center justify-center rounded-full ring-2 ring-background ${
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
                      {entry.classification !== "baseline" && (
                        <span className="text-xs text-muted-foreground">
                          {(entry.confidence * 100).toFixed(0)}% confidence
                        </span>
                      )}
                    </div>
                    {entry.classification === "baseline" ? (
                      <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                        <span>{new Date(entry.publishedAt).toLocaleDateString()}</span>
                      </div>
                    ) : (
                      <>
                        <p className="text-sm text-muted-foreground mt-1 line-clamp-2">
                          {entry.reasoning}
                        </p>
                        <div className="flex items-center gap-3 mt-1 text-xs text-muted-foreground">
                          <span>{entry.analyzerType} / {entry.modelUsed}</span>
                          <span>{new Date(entry.analyzedAt).toLocaleDateString()}</span>
                        </div>
                      </>
                    )}
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
