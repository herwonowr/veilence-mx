"use client"

import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { lazy, Suspense, useState } from "react"

// SEC-S4-10: Only load ReactQueryDevtools in development via dynamic import.
// This ensures the devtools bundle is completely excluded from production builds.
const ReactQueryDevtools =
  process.env.NODE_ENV === "development"
    ? lazy(() =>
        import("@tanstack/react-query-devtools").then((mod) => ({
          default: mod.ReactQueryDevtools,
        }))
      )
    : null

function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60 * 1000, // 1 minute
        refetchOnWindowFocus: false,
        retry: (failureCount, error) => {
          // SEC-S3-006: Don't retry on rate-limit or auth errors
          if (error instanceof Error && error.message.startsWith("Rate limited")) {
            return false
          }
          return failureCount < 1
        },
        retryDelay: (attemptIndex) =>
          Math.min(1000 * 2 ** attemptIndex, 30000), // Exponential backoff
      },
      mutations: {
        retry: 0,
      },
    },
  })
}

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(() => makeQueryClient())

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      {ReactQueryDevtools && (
        <Suspense fallback={null}>
          <ReactQueryDevtools initialIsOpen={false} />
        </Suspense>
      )}
    </QueryClientProvider>
  )
}
