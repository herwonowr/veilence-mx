import { describe, it, expect } from "vitest"
import {
  formatDownloadCount,
  formatPopularity,
  popularityLabel,
  formatFreshness,
} from "@/domains/packages/utils/packages.utils"

describe("formatDownloadCount", () => {
  it("returns em-dash for null", () => {
    expect(formatDownloadCount(null)).toBe("-")
  })

  it("returns em-dash for undefined", () => {
    expect(formatDownloadCount(undefined)).toBe("-")
  })

  it("formats billions", () => {
    expect(formatDownloadCount(1_900_000_000)).toBe("1.9B/mo")
  })

  it("formats millions", () => {
    expect(formatDownloadCount(12_500_000)).toBe("12.5M/mo")
  })

  it("formats thousands", () => {
    expect(formatDownloadCount(5_400)).toBe("5.4K/mo")
  })

  it("formats small numbers", () => {
    expect(formatDownloadCount(42)).toBe("42/mo")
  })

  it("formats zero", () => {
    expect(formatDownloadCount(0)).toBe("0/mo")
  })

  it("formats exact boundary at 1B", () => {
    expect(formatDownloadCount(1_000_000_000)).toBe("1.0B/mo")
  })

  it("formats exact boundary at 1M", () => {
    expect(formatDownloadCount(1_000_000)).toBe("1.0M/mo")
  })

  it("formats exact boundary at 1K", () => {
    expect(formatDownloadCount(1_000)).toBe("1.0K/mo")
  })
})

describe("formatPopularity", () => {
  describe("NPM ecosystem", () => {
    it("returns formatted download count when available", () => {
      expect(formatPopularity("npm", 2_700_000_000)).toBe("2.7B/mo")
    })

    it("returns formatted download count for millions", () => {
      expect(formatPopularity("npm", 633_000_000)).toBe("633.0M/mo")
    })

    it("returns em-dash when downloadCount is null", () => {
      expect(formatPopularity("npm", null)).toBe("—")
    })

    it("returns em-dash when downloadCount is undefined", () => {
      expect(formatPopularity("npm", undefined)).toBe("—")
    })

    it("formats zero downloads", () => {
      expect(formatPopularity("npm", 0)).toBe("0/mo")
    })
  })

  describe("Python ecosystem", () => {
    it("returns formatted download count", () => {
      expect(formatPopularity("python", 12_500_000)).toBe("12.5M/mo")
    })

    it("returns em-dash when downloadCount is null", () => {
      expect(formatPopularity("python", null)).toBe("—")
    })

    it("returns em-dash when downloadCount is undefined", () => {
      expect(formatPopularity("python", undefined)).toBe("—")
    })

    it("formats zero downloads", () => {
      expect(formatPopularity("python", 0)).toBe("0/mo")
    })
  })

  describe("edge cases", () => {
    it("handles unknown ecosystem with download count", () => {
      expect(formatPopularity("unknown", 1000)).toBe("1.0K/mo")
    })

    it("handles unknown ecosystem with no data", () => {
      expect(formatPopularity("unknown", undefined)).toBe("—")
    })
  })
})

describe("popularityLabel", () => {
  it("returns 'Downloads (30-day)' for npm", () => {
    expect(popularityLabel("npm")).toBe("Downloads (30-day)")
  })

  it("returns 'Downloads (30-day)' for python", () => {
    expect(popularityLabel("python")).toBe("Downloads (30-day)")
  })

  it("returns downloads label for unknown ecosystem", () => {
    expect(popularityLabel("unknown")).toBe("Downloads (30-day)")
  })
})

describe("formatFreshness", () => {
  it("returns 'Never updated' for null", () => {
    expect(formatFreshness(null)).toBe("Never updated")
  })

  it("returns 'Updated just now' for very recent date", () => {
    const now = new Date().toISOString()
    expect(formatFreshness(now)).toBe("Updated just now")
  })

  it("returns minutes ago for recent date", () => {
    const fiveMinAgo = new Date(Date.now() - 5 * 60_000).toISOString()
    expect(formatFreshness(fiveMinAgo)).toBe("Updated 5m ago")
  })

  it("returns hours ago", () => {
    const threeHoursAgo = new Date(Date.now() - 3 * 3_600_000).toISOString()
    expect(formatFreshness(threeHoursAgo)).toBe("Updated 3h ago")
  })

  it("returns days ago", () => {
    const twoDaysAgo = new Date(Date.now() - 2 * 86_400_000).toISOString()
    expect(formatFreshness(twoDaysAgo)).toBe("Updated 2d ago")
  })
})
