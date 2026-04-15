import { describe, it, expect } from "vitest"
import { formatEcosystem } from "@/domains/common/utils/common.utils"

describe("formatEcosystem", () => {
  it("formats 'python' as 'Python'", () => {
    expect(formatEcosystem("python")).toBe("Python")
  })

  it("formats 'npm' as 'NPM'", () => {
    expect(formatEcosystem("npm")).toBe("NPM")
  })

  it("passes through unknown ecosystem values unchanged", () => {
    expect(formatEcosystem("unknown")).toBe("unknown")
  })

  it("passes through arbitrary string values unchanged", () => {
    expect(formatEcosystem("cargo")).toBe("cargo")
  })

  it("passes through empty string unchanged", () => {
    expect(formatEcosystem("")).toBe("")
  })
})
