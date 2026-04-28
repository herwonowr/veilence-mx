import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
import type { SetupRequest, SetupResponse } from "@/domains/setup/types/setup.types"

export const apiInitializeSetup = async (
  data: SetupRequest
): Promise<ApiResponse<SetupResponse>> =>
  fetchApi<SetupResponse>("/api/setup/initialize", {
    method: "POST",
    body: JSON.stringify(data),
    skipAuth: true,
  })
