"use client"

import { useCallback, useMemo } from "react"
import { useRouter, useSearchParams, usePathname } from "next/navigation"
import type { SortingState } from "@tanstack/react-table"

/**
 * Syncs TanStack Table sorting state with URL search params.
 *
 * URL format: ?sort=columnId&order=asc|desc
 * On page load: reads sort/order from URL and returns initial SortingState.
 * On sort change: updates URL params (replaces history entry for same-page sort,
 * pushes for back-button support).
 */
export function useSortParams(): [SortingState, (updater: SortingState | ((prev: SortingState) => SortingState)) => void] {
  const router = useRouter()
  const pathname = usePathname()
  const searchParams = useSearchParams()

  const sorting: SortingState = useMemo(() => {
    const sort = searchParams.get("sort")
    const order = searchParams.get("order")
    if (sort) {
      return [{ id: sort, desc: order === "desc" }]
    }
    return []
  }, [searchParams])

  const setSorting = useCallback(
    (updater: SortingState | ((prev: SortingState) => SortingState)) => {
      const next = typeof updater === "function" ? updater(sorting) : updater
      const params = new URLSearchParams(searchParams.toString())

      if (next.length > 0) {
        params.set("sort", next[0].id)
        params.set("order", next[0].desc ? "desc" : "asc")
      } else {
        params.delete("sort")
        params.delete("order")
      }

      const qs = params.toString()
      const url = qs ? `${pathname}?${qs}` : pathname
      router.push(url, { scroll: false })
    },
    [sorting, searchParams, pathname, router]
  )

  return [sorting, setSorting]
}
