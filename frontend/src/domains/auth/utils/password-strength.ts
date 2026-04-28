export interface PasswordStrengthResult {
  label: string
  color: string
  width: string
}

/**
 * Calculates password strength based on length and character diversity.
 * Scoring:
 *   - length >= 8:  +1
 *   - length >= 12: +1 (rewards longer passwords per NIST)
 *   - has uppercase: +1
 *   - has digit:     +1
 *   - has special:   +1
 *
 * Returns null when the password is empty.
 */
export const getPasswordStrength = (
  password: string,
): PasswordStrengthResult | null => {
  if (!password) return null

  let score = 0
  if (password.length >= 8) score++
  if (password.length >= 12) score++
  if (/[A-Z]/.test(password)) score++
  if (/[0-9]/.test(password)) score++
  if (/[^A-Za-z0-9]/.test(password)) score++

  if (score <= 1) return { label: "Weak", color: "bg-red-500", width: "w-1/4" }
  if (score <= 3) return { label: "Medium", color: "bg-yellow-500", width: "w-2/4" }
  return { label: "Strong", color: "bg-green-500", width: "w-full" }
}
