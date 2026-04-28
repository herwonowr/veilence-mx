/**
 * Error message sanitization utility.
 *
 * Maps raw API / network errors to user-friendly strings so that
 * stack traces, internal status codes, or database details are never
 * shown in the UI.
 */

/** Patterns matched against Error.message (case-insensitive). */
const ERROR_PATTERNS: ReadonlyArray<{ pattern: RegExp; friendly: string }> = [
  // Auth / session errors
  { pattern: /unauthorized/i, friendly: "Your session has expired. Please sign in again." },
  { pattern: /forbidden/i, friendly: "You do not have permission to perform this action." },
  { pattern: /invalid credentials/i, friendly: "Invalid email or password." },
  { pattern: /invalid email or password/i, friendly: "Invalid email or password." },
  { pattern: /account.*disabled/i, friendly: "This account has been disabled. Contact your administrator." },
  { pattern: /email.*already/i, friendly: "An account with this email already exists." },
  { pattern: /invalid or expired reset token/i, friendly: "This reset link is invalid or has expired. Please request a new one." },
  { pattern: /token.*expired/i, friendly: "Your session has expired. Please sign in again." },
  { pattern: /invalid.*token/i, friendly: "Your session is invalid. Please sign in again." },
  { pattern: /must_change_password/i, friendly: "You must change your password before continuing." },

  // Rate limiting (already user-friendly from api-client, but catch the fallback)
  { pattern: /rate limit/i, friendly: "Too many requests. Please wait a moment and try again." },

  // Network / connectivity
  { pattern: /failed to fetch/i, friendly: "Unable to connect to the server. Check your network connection." },
  { pattern: /network/i, friendly: "A network error occurred. Please check your connection and try again." },
  { pattern: /timeout/i, friendly: "The request timed out. Please try again." },
  { pattern: /abort/i, friendly: "The request was cancelled." },

  // Server errors
  { pattern: /5\d{2}$/,  friendly: "An unexpected server error occurred. Please try again later." },
  { pattern: /API error: 5\d{2}/i, friendly: "An unexpected server error occurred. Please try again later." },
  { pattern: /API error: 4\d{2}/i, friendly: "The request could not be completed. Please try again." },

  // Validation - place AFTER unsafe checks would occur in isSafeMessage fallback
  { pattern: /bad request/i, friendly: "The request was invalid. Please check your input." },
  { pattern: /conflict/i, friendly: "A conflict occurred. The resource may have been modified by someone else." },
  { pattern: /payload too large/i, friendly: "The uploaded data is too large." },
]

const extractRawMessage = (error: unknown): string => {
  if (error instanceof Error) return error.message
  if (typeof error === "string") return error
  return "Unknown error"
}

/**
 * Heuristic: a message is considered "safe" if it is short, does not contain
 * stack-trace keywords, file paths, or SQL fragments.
 */
const isSafeMessage = (msg: string): boolean => {
  if (msg.length > 200) return false

  const unsafePatterns = [
    /at\s+\w+\s+\(/i,         // stack trace line
    /\.go:\d+/,                // Go source reference
    /\.ts:\d+/,                // TS source reference
    /\.js:\d+/,                // JS source reference
    /sql:|select\s|insert\s/i, // SQL fragments
    /panic:/i,                 // Go panic
    /GORM/i,                   // ORM internals
    /pq:/i,                    // Postgres driver
    /redis:/i,                 // Redis driver
    /internal server/i,        // generic internal
    /stack trace/i,
  ]

  return !unsafePatterns.some((p) => p.test(msg))
}

/** Capitalize the first letter of a string for user-facing display. */
const capitalizeFirst = (s: string): string =>
  s.length === 0 ? s : s.charAt(0).toUpperCase() + s.slice(1)

/**
 * Sanitise an error into a user-friendly message.
 *
 * @param error - The caught error (Error, string, or unknown).
 * @param fallback - Optional fallback message. Defaults to "Something went wrong. Please try again."
 * @returns A safe, user-facing message string with the first letter capitalized.
 */
export const sanitizeErrorMessage = (error: unknown, fallback?: string): string => {
  const raw = extractRawMessage(error)

  for (const { pattern, friendly } of ERROR_PATTERNS) {
    if (pattern.test(raw)) {
      return friendly
    }
  }

  // Fallback: if the message looks "safe" (short, no stack-trace markers),
  // pass it through with capitalized first letter (backend sends lowercase).
  // Otherwise return the provided fallback or a generic message.
  if (isSafeMessage(raw)) {
    return capitalizeFirst(raw)
  }

  return fallback ?? "Something went wrong. Please try again."
}
