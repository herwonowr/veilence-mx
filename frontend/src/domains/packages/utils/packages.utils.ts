import type { Ecosystem } from "@/domains/common"

/**
 * Format a number with human-readable suffix and the given unit tag.
 */
const formatCountWithSuffix = (
  count: number | undefined | null,
  suffix: string
): string => {
  if (count == null) return "-"
  if (count >= 1_000_000_000) {
    return `${(count / 1_000_000_000).toFixed(1)}B${suffix}`
  }
  if (count >= 1_000_000) {
    return `${(count / 1_000_000).toFixed(1)}M${suffix}`
  }
  if (count >= 1_000) {
    return `${(count / 1_000).toFixed(1)}K${suffix}`
  }
  return `${count}${suffix}`
}

/**
 * Format download count as human-readable string.
 * Works for both PyPI and NPM: "12.5M/m", "1.2K/m", etc.
 */
const formatDownloadCount = (count: number | undefined | null): string =>
  formatCountWithSuffix(count, "/m")

/**
 * Format star count as human-readable string for Go ecosystem.
 * E.g., "94.9K/s", "1.2M/s".
 */
const formatStarCount = (count: number | undefined | null): string =>
  formatCountWithSuffix(count, "/s")

/**
 * Format the popularity metric for display.
 * Go ecosystem uses GitHub stars (/s suffix).
 * PyPI and NPM show formatted download counts (/m suffix).
 * Returns "-" when no count is available.
 */
export const formatPopularity = (
  ecosystem: Ecosystem | string,
  downloadCount: number | undefined | null
): string => {
  if (downloadCount == null) return "-"
  if (ecosystem === "go") return formatStarCount(downloadCount)
  return formatDownloadCount(downloadCount)
}

/**
 * Returns a full tooltip string for the popularity cell including the metric
 * name and freshness timestamp.
 * e.g. "94.9K GitHub stars - Updated 21m ago" or "12.5M monthly downloads - Updated 2h ago"
 */
export const popularityTooltip = (
  ecosystem: Ecosystem | string,
  downloadCount: number | undefined | null,
  downloadCountUpdatedAt: string | null,
): string => {
  if (downloadCount == null) return "No data available"
  const formatted = formatCountWithSuffix(downloadCount, "")
  const metric = ecosystem === "go" ? "GitHub stars" : "monthly downloads"
  const freshness = formatFreshness(downloadCountUpdatedAt)
  return `${formatted} ${metric} - ${freshness}`
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
