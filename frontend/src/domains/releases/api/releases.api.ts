import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { ReleaseDetail, ReanalyzeReleaseResponse, PipelineStatus } from "@/domains/releases/types/releases.types"

export const getRelease = async (
  id: string
): Promise<ApiResponse<ReleaseDetail>> =>
  fetchApi<ReleaseDetail>(`/api/releases/${id}`)

export const reanalyzeRelease = async (
  releaseId: string
): Promise<ApiResponse<ReanalyzeReleaseResponse>> =>
  fetchApi<ReanalyzeReleaseResponse>(`/api/releases/${releaseId}/reanalyze`, {
    method: "POST",
  })

export const getPipelineStatus = async (): Promise<ApiResponse<PipelineStatus>> =>
  fetchApi<PipelineStatus>(`/api/releases/pipeline-status`)

