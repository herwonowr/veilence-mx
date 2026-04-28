import { fetchApi } from "@/core"
import type { ApiResponse } from "@/domains/common"
// Pragmatic exception: intra-domain deep import to avoid circular dependency via barrel
import type { SetupRequest, SetupResponse } from "@/domains/setup/types/setup.types"

export const apiInitializeSetup = async (
  data: SetupRequest
): Promise<ApiResponse<SetupResponse>> =>
  fetchApi<SetupResponse>("/api/setup/initialize", {
    method: "POST",
    body: JSON.stringify(data),
    skipAuth: true,
  })
