"use client"

import { useState, useCallback } from "react"
import { apiInitializeSetup } from "@/domains/setup"
import type { SetupRequest, SetupResponse } from "@/domains/setup"
import { sanitizeErrorMessage } from "@/core"

export const useSetup = () => {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [data, setData] = useState<SetupResponse | null>(null)

  const initialize = useCallback(async (request: SetupRequest) => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await apiInitializeSetup(request)
      setData(response.data)
      return response.data
    } catch (err: unknown) {
      const message = sanitizeErrorMessage(err, "Setup failed")
      setError(message)
      throw err
    } finally {
      setIsLoading(false)
    }
  }, [])

  return { initialize, isLoading, error, data }
}
