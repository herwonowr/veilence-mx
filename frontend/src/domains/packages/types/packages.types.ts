import type { Ecosystem, ReleaseStatus, Classification, AnalyzerType } from "@/domains/common"

export type PackageSource = "manual" | "discovered" | "imported"
export type PackageStatus = "active" | "suggested" | "blocked" | "removed"

export interface Package {
  id: string
  name: string
  ecosystem: Ecosystem
  latestVersion: string
  description: string
  source: PackageSource
  status: PackageStatus
  downloadCount: number
  workspaceId: string
  downloadCountUpdatedAt: string | null
  blockedAt: string | null
  blockedReason: string | null
  createdAt: string
  updatedAt: string
}

export interface StalePackage extends Package {
  lastReleaseAt: string | null
  daysSinceLastRelease: number
}

export interface BulkImportError {
  name: string
  error: string
}

export interface BulkImportResult {
  imported: number
  skipped: number
  errors: BulkImportError[]
}

export interface AnalysisHistoryEntry {
  releaseId: string
  version: string
  classification: Classification
  confidence: number
  reasoning: string
  modelUsed: string
  analyzerType: AnalyzerType
  analyzedAt: string
  publishedAt: string
}

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
