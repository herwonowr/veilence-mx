"use client"

import { use } from "react"
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
import { Skeleton } from "@/components/ui/skeleton"
import { ArrowLeft } from "lucide-react"
import { ProtectedRoute } from "@/components/protected-route"
import { usePackage, usePackageReleases } from "@/features/packages"

export default function PackageDetailPage({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  return (
    <ProtectedRoute>
      <PackageDetailContent params={params} />
    </ProtectedRoute>
  )
}

function PackageDetailContent({
  params,
}: {
  params: Promise<{ id: string }>
}) {
  const { id } = use(params)
  const packageId = parseInt(id, 10)

  const { data: pkgRes } = usePackage(packageId)
  const { data: releasesRes } = usePackageReleases(packageId)

  const pkg = pkgRes?.data ?? null
  const releases = releasesRes?.data ?? []

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
        <div className="flex items-center gap-2 mt-2">
          <Badge variant="outline">{pkg.registry}</Badge>
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
                <TableRow key={release.id}>
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
                  <TableCell colSpan={3} className="text-center text-muted-foreground py-8">
                    No releases tracked yet.
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
