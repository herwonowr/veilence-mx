import type { Classification, AnalyzerType, ReleaseStatus } from "@/domains/common"

export interface Release {
  id: number
  packageId: number
  version: string
  publishedAt: string
  tarballUrl: string
  sha256: string
  status: ReleaseStatus
  errorMessage?: string
  createdAt: string
}

export interface Diff {
  id: number
  releaseId: number
  prevReleaseId: number
  diffContent: string
  fileChangesCount: number
  linesAdded: number
  linesRemoved: number
  createdAt: string
}

export interface Analysis {
  id: number
  diffId: number
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
    id: number
    name: string
    ecosystem: string
    latestVersion: string
    description: string
    source: string
    status: string
    blockedAt: string | null
    blockedReason: string | null
    downloadCount: number
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
