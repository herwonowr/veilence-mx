import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/** Format ecosystem API value for display: npm→NPM, python→Python */
export function formatEcosystem(value: string): string {
  if (value === "npm") return "NPM"
  if (value === "python") return "Python"
  return value.charAt(0).toUpperCase() + value.slice(1)
}
