"use client"

import { useState, useCallback } from "react"
import { apiGetPublicConfig } from "@/domains/config"
import type { PublicConfig } from "@/domains/config"
import { sanitizeErrorMessage } from "@/core"

export const usePublicConfig = () => {
  const [config, setConfig] = useState<PublicConfig | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchConfig = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await apiGetPublicConfig()
      setConfig(response.data)
      return response.data
    } catch (err: unknown) {
      const message = sanitizeErrorMessage(err, "Failed to load configuration")
      setError(message)
      return null
    } finally {
      setIsLoading(false)
    }
  }, [])

  return { config, isLoading, error, fetchConfig }
}
