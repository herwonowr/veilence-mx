"use client"

import { use } from "react"
import Link from "next/link"
import { notFound } from "next/navigation"
import { Card, CardContent, Badge, Button, Skeleton, DetailError } from "@/ui"
import { ArrowLeft, ExternalLink } from "lucide-react"
import { formatEcosystem } from "@/domains/common"
import { useCurrentWorkspaceRole, hasMinimumRole } from "@/core"
import { useAlert, useUpdateAlert } from "@/features/alerts/hooks/use-alerts"
import { AlertNotesTimeline } from "@/features/alerts/ui/alert-notes-timeline"
import type { AlertSeverity } from "@/domains/common"

const severityVariant = (s: AlertSeverity) => {
  if (s === "critical") return "destructive" as const
  if (s === "high") return "destructive" as const
  if (s === "medium") return "default" as const
  return "secondary" as const
}

export const AlertDetailView = ({
  params,
}: {
  params: Promise<{ id: string }>
}) => {
  const { id } = use(params)
  const alertId = id

  if (!alertId) {
    notFound()
  }

  const { data: alertRes, isError, refetch } = useAlert(alertId)
  const updateMutation = useUpdateAlert()
  const { role: currentRole } = useCurrentWorkspaceRole()
  const canTriage = hasMinimumRole(currentRole, "member")

  const alert = alertRes?.data ?? null

  if (isError) return (
    <DetailError
      message="Failed to load alert details. The alert may not exist or the server is unavailable."
      onRetry={() => refetch()}
      backHref="/alerts"
      backLabel="Alerts"
    />
  )

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/alerts"
          className="w-fit text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Alerts
        </Link>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <h1 className="text-3xl font-bold">Alert #{alertId}</h1>
          {alert && canTriage && (
            <div className="flex flex-wrap items-center gap-2">
              {alert.status === "new" && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => updateMutation.mutate({ id: alertId, status: "acknowledged" })}
                >
                  Acknowledge
                </Button>
              )}
              {alert.status !== "resolved" && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => updateMutation.mutate({ id: alertId, status: "resolved" })}
                >
                  Resolve
                </Button>
              )}
            </div>
          )}
        </div>
      </div>

      {alert ? (
        <Card>
          <CardContent className="pt-6 space-y-4">
            <div className="flex flex-wrap items-center gap-3">
              <Badge variant={severityVariant(alert.severity)} className="text-sm">
                {alert.severity}
              </Badge>
              <Badge variant="outline">{alert.status}</Badge>
              <span className="text-sm text-muted-foreground">
                {new Date(alert.createdAt).toLocaleString()}
              </span>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Package</p>
              <Link
                href={`/packages/${alert.packageId}`}
                className="text-primary hover:underline font-medium"
              >
                {alert.packageName}
                <span className="text-xs text-muted-foreground ml-1">({formatEcosystem(alert.packageEcosystem ?? "")})</span>
              </Link>
            </div>
            <div>
              <p className="text-sm font-medium text-muted-foreground">Message</p>
              <p className="text-sm leading-relaxed mt-1">{alert.message}</p>
            </div>
            <div>
              {(alert.releaseId ?? alert.analysisId) ? (
                <Link
                  href={`/releases/${alert.releaseId ?? alert.analysisId}`}
                  className="inline-flex items-center gap-1 text-sm text-primary hover:underline"
                >
                  <ExternalLink className="h-3.5 w-3.5" />
                  View Release Analysis
                </Link>
              ) : (
                <span className="inline-flex items-center gap-1 text-sm text-muted-foreground">
                  <ExternalLink className="h-3.5 w-3.5" />
                  No release linked
                </span>
              )}
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="pt-6 space-y-4">
            <Skeleton className="h-6 w-32" />
            <Skeleton className="h-4 w-64" />
            <Skeleton className="h-16 w-full" />
          </CardContent>
        </Card>
      )}

      {/* Notes Timeline */}
      <AlertNotesTimeline alertId={alertId} />
    </div>
  )
}
