import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { ReleaseDetail, ReanalyzeReleaseResponse } from "@/domains/releases/types/releases.types"

export const getRelease = async (
  id: number
): Promise<ApiResponse<ReleaseDetail>> =>
  fetchApi<ReleaseDetail>(`/api/releases/${id}`)

export const reanalyzeRelease = async (
  releaseId: number
): Promise<ApiResponse<ReanalyzeReleaseResponse>> =>
  fetchApi<ReanalyzeReleaseResponse>(`/api/releases/${releaseId}/reanalyze`, {
    method: "POST",
  })

