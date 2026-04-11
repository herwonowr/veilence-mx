import { useEffect, useState } from "react"

/**
 * Returns a debounced version of the input value.
 * The returned value only updates after `delay` ms of no changes to the input.
 *
 * Useful for delaying API requests triggered by search inputs.
 */
export function useDebouncedValue<T>(value: T, delay = 300): T {
  const [debouncedValue, setDebouncedValue] = useState(value)

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedValue(value)
    }, delay)

    return () => clearTimeout(timer)
  }, [value, delay])

  return debouncedValue
}
