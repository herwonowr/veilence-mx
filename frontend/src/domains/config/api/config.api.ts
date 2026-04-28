import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { PublicConfig } from "@/domains/config/types/config.types"

export const apiGetPublicConfig = async (): Promise<ApiResponse<PublicConfig>> =>
  fetchApi<PublicConfig>("/api/config/public", { skipAuth: true })
