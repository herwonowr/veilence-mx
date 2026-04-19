"use client"

import { useState, useCallback, useEffect } from "react"

export const useLocalStorage = <T>(key: string, defaultValue: T): [T, (value: T | ((prev: T) => T)) => void] => {
  const [storedValue, setStoredValue] = useState<T>(() => {
    if (typeof window === "undefined") return defaultValue
    try {
      const item = window.localStorage.getItem(key)
      return item !== null ? (JSON.parse(item) as T) : defaultValue
    } catch {
      return defaultValue
    }
  })

  const setValue = useCallback(
    (value: T | ((prev: T) => T)) => {
      setStoredValue((prev) => {
        const nextValue = value instanceof Function ? value(prev) : value
        try {
          window.localStorage.setItem(key, JSON.stringify(nextValue))
        } catch {
          // Storage full or unavailable — silently degrade
        }
        return nextValue
      })
    },
    [key]
  )

  // Sync across tabs
  useEffect(() => {
    const handler = (e: StorageEvent) => {
      if (e.key !== key) return
      try {
        setStoredValue(e.newValue !== null ? (JSON.parse(e.newValue) as T) : defaultValue)
      } catch {
        setStoredValue(defaultValue)
      }
    }
    window.addEventListener("storage", handler)
    return () => window.removeEventListener("storage", handler)
  }, [key, defaultValue])

  return [storedValue, setValue]
}
