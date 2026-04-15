import type { Ecosystem } from "@/domains/common"

/**
 * Format download count for PyPI packages as human-readable string.
 * PyPI: "12.5M/mo", "1.2K/mo", etc.
 * NPM: shows popularity score instead (handled separately).
 */
export const formatDownloadCount = (count: number | undefined | null): string => {
  if (count == null) return "—"
  if (count >= 1_000_000_000) {
    return `${(count / 1_000_000_000).toFixed(1)}B/mo`
  }
  if (count >= 1_000_000) {
    return `${(count / 1_000_000).toFixed(1)}M/mo`
  }
  if (count >= 1_000) {
    return `${(count / 1_000).toFixed(1)}K/mo`
  }
  return `${count}/mo`
}

/**
 * Format the popularity metric for display based on ecosystem.
 * PyPI: formatted download count (e.g., "12.5M/mo")
 * NPM: popularity score (e.g., "0.95")
 */
export const formatPopularity = (
  ecosystem: Ecosystem | string,
  downloadCount: number | undefined | null,
  popularityScore: number | undefined | null
): string => {
  if (ecosystem === "npm") {
    if (popularityScore == null) return "—"
    return popularityScore.toFixed(2)
  }
  return formatDownloadCount(downloadCount)
}

/**
 * Returns a human-readable label for the popularity column header tooltip.
 */
export const popularityLabel = (ecosystem: Ecosystem | string): string => {
  if (ecosystem === "npm") return "Popularity Score"
  return "Downloads (30-day)"
}

/**
 * Format a date string as relative freshness text.
 * e.g., "Updated 2 hours ago", "Updated 3 days ago"
 */
export const formatFreshness = (dateStr: string | null): string => {
  if (!dateStr) return "Never updated"
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diffMs = now - then
  const diffMin = Math.floor(diffMs / 60_000)
  const diffHours = Math.floor(diffMs / 3_600_000)
  const diffDays = Math.floor(diffMs / 86_400_000)

  if (diffMin < 1) return "Updated just now"
  if (diffMin < 60) return `Updated ${diffMin}m ago`
  if (diffHours < 24) return `Updated ${diffHours}h ago`
  return `Updated ${diffDays}d ago`
}
