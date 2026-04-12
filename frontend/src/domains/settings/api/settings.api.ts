import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { ReanalyzeAllResponse } from "@/domains/settings/types/settings.types"

export const getSettings = async (): Promise<
  ApiResponse<Record<string, string>>
> => fetchApi<Record<string, string>>("/api/settings")

export const updateSettings = async (
  settings: Record<string, string>
): Promise<ApiResponse<Record<string, string>>> =>
  fetchApi<Record<string, string>>("/api/settings", {
    method: "PUT",
    body: JSON.stringify(settings),
  })

export const reanalyzeAll = async (): Promise<
  ApiResponse<ReanalyzeAllResponse>
> =>
  fetchApi<ReanalyzeAllResponse>("/api/sync/reanalyze", {
    method: "POST",
  })
