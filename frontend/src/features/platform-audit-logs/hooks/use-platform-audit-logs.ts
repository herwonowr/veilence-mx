"use client"

import { useQuery } from "@tanstack/react-query"
import {
  apiGetPlatformAuditLogs,
  platformAdminKeys,
} from "@/domains/platform-admin"
import type { PlatformAuditLogParams } from "@/domains/platform-admin"

export const usePlatformAuditLogs = (params: PlatformAuditLogParams) =>
  useQuery({
    queryKey: platformAdminKeys.auditLogs(params as Record<string, string | number | undefined>),
    queryFn: () => apiGetPlatformAuditLogs(params),
  })
