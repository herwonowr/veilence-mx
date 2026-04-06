import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * SEC-S3-006: Sanitize error messages for user-facing toasts.
 * Shows the fallback for non-Error objects or errors that may leak internals.
 * Only passes through messages that look like intentional API error strings.
 */
export function sanitizeErrorMessage(
  error: unknown,
  fallback: string
): string {
  if (!(error instanceof Error)) return fallback
  const msg = error.message
  // Allow through: rate-limit messages, and concise API error messages (< 200 chars, no stack traces)
  if (
    msg &&
    msg.length < 200 &&
    !msg.includes("\n") &&
    !msg.includes("at ") &&
    !msg.includes("TypeError") &&
    !msg.includes("SyntaxError") &&
    !msg.includes("ReferenceError")
  ) {
    return msg
  }
  return fallback
}
