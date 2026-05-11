import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"
import { ZodError } from "zod"

export const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs))

/** Extract field-level error messages from a caught error (ZodError or unknown). */
export const parseFieldErrors = (err: unknown): Record<string, string> => {
  if (!(err instanceof ZodError)) return {}
  const fields: Record<string, string> = {}
  for (const issue of err.issues) {
    const key = issue.path[0]
    if (typeof key === "string" && !fields[key]) fields[key] = issue.message
  }
  return fields
}
