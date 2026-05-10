import type { Classification, AnalyzerType, ReleaseStatus } from "@/domains/common"

export interface Release {
  id: string
  packageId: string
  version: string
  publishedAt: string
  tarballUrl: string
  sha256: string
  status: ReleaseStatus
  errorMessage?: string
  createdAt: string
}

export interface Diff {
  id: string
  releaseId: string
  prevReleaseId: string
  diffContent: string
  fileChangesCount: number
  linesAdded: number
  linesRemoved: number
  truncated: boolean
  originalSize: number
  createdAt: string
}

export interface Analysis {
  id: string
  diffId: string
  classification: Classification
  confidence: number
  reasoning: string
  modelUsed: string
  analyzerType: AnalyzerType
  createdAt: string
}

export interface ReleaseDetail extends Release {
  diff?: Diff
  analysis?: Analysis
  package?: {
    id: string
    name: string
    ecosystem: string
    latestVersion: string
    description: string
    source: string
    status: string
    blockedAt: string | null
    blockedReason: string | null
    downloadCount: number
    workspaceId: string
    downloadCountUpdatedAt: string | null
    createdAt: string
    updatedAt: string
  }
  isBaseline?: boolean
}

export interface RecentRelease extends Release {
  packageName: string
  packageEcosystem: string
  classification?: Classification
}

export interface ReanalyzeReleaseResponse {
  message: string
  jobId: string
}
