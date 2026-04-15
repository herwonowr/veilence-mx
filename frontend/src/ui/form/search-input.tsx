"use client"

import { useRef, useCallback } from "react"
import { Search, X, Loader2 } from "lucide-react"
import { Input } from "@/ui/components/input"
import { cn } from "@/core/utils"
import { useDebouncedValue } from "@/core/hooks/use-debounced-value"

interface SearchInputProps {
  value: string
  onChange: (value: string) => void
  /** Called when user clicks the clear button - use to trigger immediate fetch */
  onClear?: () => void
  isLoading?: boolean
  placeholder?: string
  "aria-label"?: string
  className?: string
  debounceMs?: number
}

export const SearchInput = ({
  value,
  onChange,
  onClear,
  isLoading = false,
  placeholder = "Search...",
  "aria-label": ariaLabel,
  className,
  debounceMs = 300,
}: SearchInputProps) => {
  const debouncedValue = useDebouncedValue(value, debounceMs)
  const isDebouncing = !!value && value !== debouncedValue
  const inputRef = useRef<HTMLInputElement>(null)

  const handleClear = useCallback(() => {
    onChange("")
    onClear?.()
    inputRef.current?.focus()
  }, [onChange, onClear])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Escape" && value) {
        e.preventDefault()
        handleClear()
      }
    },
    [value, handleClear]
  )

  return (
    <div className={cn("relative py-1 max-w-xs w-full", className)}>
      {/* Left icon: spinner when loading, magnifying glass otherwise */}
      <div className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2" aria-hidden="true">
        {isLoading ? (
          <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
        ) : (
          <Search className="h-4 w-4 text-muted-foreground" />
        )}
      </div>

      <Input
        ref={inputRef}
        role="searchbox"
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        aria-label={ariaLabel ?? placeholder}
        className={cn(
          "pl-9 pr-8",
          isDebouncing && !isLoading && "ring-2 ring-ring/40 animate-pulse",
        )}
      />

      {/* Right: clear button when has value */}
      {value && (
        <button
          type="button"
          className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded-full p-0.5 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
          onClick={handleClear}
          aria-label="Clear search"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      )}

      {/* Screen reader announcement for loading */}
      {isLoading && (
        <span className="sr-only" aria-live="polite">
          Searching...
        </span>
      )}
    </div>
  )
}
