"use client"

import { useEffect, useRef } from "react"
import { useRouter, usePathname } from "next/navigation"

/**
 * Syncs a filter state record to URL search params.
 *
 * - When a filter value equals its default (or is empty string), the param is REMOVED from the URL.
 * - Uses `router.replace` with `{ scroll: false }` to avoid page reload and scroll reset.
 * - Skips the initial render to avoid overwriting the URL on mount (the URL is the source of truth on mount).
 *
 * @param filters - Current filter values keyed by param name
 * @param defaults - Default/empty values per key. When a filter matches its default, the param is omitted.
 */
export const useFilterParams = (
  filters: Record<string, string>,
  defaults?: Record<string, string>,
) => {
  const router = useRouter()
  const pathname = usePathname()
  const isInitialMount = useRef(true)

  useEffect(() => {
    // Skip the first render — on mount the URL is already the source of truth
    if (isInitialMount.current) {
      isInitialMount.current = false
      return
    }

    const params = new URLSearchParams()

    for (const [key, value] of Object.entries(filters)) {
      const defaultValue = defaults?.[key] ?? ""
      if (value && value !== defaultValue) {
        params.set(key, value)
      }
    }

    const search = params.toString()
    const url = search ? `${pathname}?${search}` : pathname
    router.replace(url, { scroll: false })
  }, [filters, defaults, pathname, router])
}
