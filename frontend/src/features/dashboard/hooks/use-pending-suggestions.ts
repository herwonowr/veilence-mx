"use client"

import { useQuery } from "@tanstack/react-query"
import { getPackageSuggestions } from "@/domains/packages"

export const pendingSuggestionsKeys = {
  count: ["dashboard", "pending-suggestions-count"] as const,
}

export const usePendingSuggestionsCount = () =>
  useQuery({
    queryKey: pendingSuggestionsKeys.count,
    queryFn: async () => {
      const res = await getPackageSuggestions({ limit: 1 })
      return res.meta?.total ?? 0
    },
    staleTime: 30 * 1000,
  })
