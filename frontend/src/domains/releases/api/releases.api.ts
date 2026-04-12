import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { ReleaseDetail } from "@/domains/releases/types/releases.types"

export const getRelease = async (
  id: number
): Promise<ApiResponse<ReleaseDetail>> =>
  fetchApi<ReleaseDetail>(`/api/releases/${id}`)

export const reanalyzeRelease = async (
  releaseId: number
): Promise<ApiResponse<{ message: string }>> =>
  fetchApi<{ message: string }>(`/api/releases/${releaseId}/reanalyze`, {
    method: "POST",
  })

