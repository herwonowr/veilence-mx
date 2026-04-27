"use client"

import { use, useState } from "react"
import Link from "next/link"
import { notFound } from "next/navigation"
import { Card, CardContent, CardHeader, CardTitle, Badge, Separator, Skeleton, Button, DetailError, Progress } from "@/ui"
import type { Classification } from "@/domains/common"
import { ArrowLeft, FileCode, Plus, Minus, WrapText, RotateCcw, Loader2 } from "lucide-react"
import { formatEcosystem } from "@/domains/common"
import { useCurrentWorkspaceRole, hasMinimumRole } from "@/core"
import { useRelease, useReanalyzeRelease } from "@/features/releases/hooks/use-releases"

const classificationColor = (c: Classification) => {
  if (c === "malicious") return "destructive" as const
  if (c === "suspicious") return "default" as const
  return "secondary" as const
}

const confidenceColor = (confidence: number): string => {
  if (confidence >= 0.8) return "[&_[data-slot=progress-indicator]]:bg-green-500"
  if (confidence >= 0.5) return "[&_[data-slot=progress-indicator]]:bg-amber-500"
  return "[&_[data-slot=progress-indicator]]:bg-red-500"
}

const DiffViewer = ({ content, wordWrap }: { content: string; wordWrap: boolean }) => {
  const lines = content.split("\n")
  return (
    <div className="overflow-auto rounded-lg border text-xs font-mono max-h-175" role="region" aria-label="Diff viewer" tabIndex={0}>
      {lines.map((line, i) => {
        let bgClass = ""
        let textClass = "text-foreground"
        if (line.startsWith("+") && !line.startsWith("+++")) {
          bgClass = "bg-green-500/10"
          textClass = "text-green-700 dark:text-green-400"
        } else if (line.startsWith("-") && !line.startsWith("---")) {
          bgClass = "bg-red-500/10"
          textClass = "text-red-700 dark:text-red-400"
        } else if (line.startsWith("@@")) {
          bgClass = "bg-blue-500/10"
          textClass = "text-blue-700 dark:text-blue-400"
        } else if (line.startsWith("diff ") || line.startsWith("index ") || line.startsWith("---") || line.startsWith("+++")) {
          textClass = "text-muted-foreground font-semibold"
        }
        return (
          <div key={i} className={`px-4 py-0 leading-5 ${wordWrap ? "whitespace-pre-wrap break-all" : "whitespace-pre"} ${bgClass} ${textClass}`}>
            {line || " "}
          </div>
        )
      })}
    </div>
  )
}

export const ReleaseDetailView = ({
  params,
}: {
  params: Promise<{ id: string }>
}) => {
  const { id } = use(params)
  const releaseId = id
  const [wordWrap, setWordWrap] = useState(false)

  // Validation: show 404 for empty IDs
  if (!releaseId) {
    notFound()
  }

  const { data: releaseRes, isError, refetch } = useRelease(releaseId)
  const reanalyzeMutation = useReanalyzeRelease(releaseId)
  const { role: currentRole } = useCurrentWorkspaceRole()
  const canReanalyze = hasMinimumRole(currentRole, "admin")
  const release = releaseRes?.data ?? null

  if (isError) return (
    <DetailError
      message="Failed to load release details. The release may not exist or the server is unavailable."
      onRetry={() => refetch()}
      backHref="/releases"
      backLabel="Releases"
    />
  )

  if (!release) return (
    <div className="space-y-6 p-8">
      <Skeleton className="h-4 w-32" />
      <Skeleton className="h-8 w-64" />
      <div className="flex gap-2">
        <Skeleton className="h-6 w-16" />
        <Skeleton className="h-6 w-20" />
      </div>
      <Skeleton className="h-96" />
    </div>
  )

  return (
    <div className="space-y-6 min-w-0">
      <div>
        {release.package && (
          <Link
            href={`/packages/${release.package.id}`}
            className="w-fit text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            {release.package.name}
          </Link>
        )}
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <h1 className="text-3xl font-bold">
            {release.package?.name} <span className="text-muted-foreground font-normal">v{release.version}</span>
          </h1>
          {/* Re-analyze button */}
          {canReanalyze && release.status === "completed" && !release.isBaseline && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => reanalyzeMutation.mutate()}
              disabled={reanalyzeMutation.isPending}
              aria-label="Re-analyze this release"
            >
              {reanalyzeMutation.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <RotateCcw className="mr-2 h-4 w-4" />
              )}
              Re-analyze
            </Button>
          )}
        </div>
        <div className="flex flex-wrap items-center gap-2 mt-2">
          <Badge variant="outline">{formatEcosystem(release.package?.ecosystem ?? "")}</Badge>
          <Badge variant="secondary">{release.status}</Badge>
          <span className="text-sm text-muted-foreground">
            Published {new Date(release.publishedAt).toLocaleDateString()}
          </span>
        </div>
      </div>

      {release.analysis && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              Analysis Result
              <Badge variant={classificationColor(release.analysis.classification)}>
                {release.analysis.classification}
              </Badge>
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-1">
                <p className="text-sm font-medium text-muted-foreground">Confidence</p>
                <span className="text-sm font-bold">
                  {(release.analysis.confidence * 100).toFixed(0)}%
                </span>
              </div>
              <Progress
                value={Math.round(release.analysis.confidence * 100)}
                aria-label="Analysis confidence"
                className={confidenceColor(release.analysis.confidence)}
              />
            </div>
            <Separator />
            <div>
              <p className="text-sm font-medium text-muted-foreground mb-1">Reasoning</p>
              <p className="text-sm leading-relaxed">{release.analysis.reasoning}</p>
            </div>
            <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
              <span>Analyzer: <strong>{release.analysis.analyzerType}</strong></span>
              <span>Model: <strong>{release.analysis.modelUsed}</strong></span>
              <span>Analyzed: {new Date(release.analysis.createdAt).toLocaleString()}</span>
            </div>
          </CardContent>
        </Card>
      )}

      {!release.analysis && release.status === "completed" && release.isBaseline && (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            <Badge variant="outline" className="mb-2">baseline</Badge>
            <p>This is the first tracked version - no previous release to compare against.</p>
          </CardContent>
        </Card>
      )}

      {!release.analysis && release.diff && release.status === "completed" && (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            Analysis not available - the LLM analysis may have failed or is still queued for retry.
          </CardContent>
        </Card>
      )}

      {!release.analysis && !release.diff && release.status === "completed" && !release.isBaseline && (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            No diff or analysis available for this release.
          </CardContent>
        </Card>
      )}

      {release.diff && (
        <Card className="min-w-0 overflow-hidden">
          <CardHeader>
            <CardTitle className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex flex-wrap items-center gap-4">
                <span className="flex items-center gap-2">
                  <FileCode className="h-5 w-5" />
                  Diff
                </span>
                <div className="flex items-center gap-3 text-sm font-normal text-muted-foreground">
                  <span>{release.diff.fileChangesCount} files</span>
                  <span className="flex items-center gap-0.5 text-green-600">
                    <Plus className="h-3.5 w-3.5" />
                    {release.diff.linesAdded.toLocaleString()}
                  </span>
                  <span className="flex items-center gap-0.5 text-red-600">
                    <Minus className="h-3.5 w-3.5" />
                    {release.diff.linesRemoved.toLocaleString()}
                  </span>
                </div>
              </div>
              <Button
                variant={wordWrap ? "secondary" : "ghost"}
                size="sm"
                onClick={() => setWordWrap((v) => !v)}
                title={wordWrap ? "Disable word wrap" : "Enable word wrap"}
                aria-label={wordWrap ? "Disable word wrap" : "Enable word wrap"}
                aria-pressed={wordWrap}
              >
                <WrapText className="h-4 w-4" />
              </Button>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <DiffViewer content={release.diff.diffContent} wordWrap={wordWrap} />
          </CardContent>
        </Card>
      )}
    </div>
  )
}
