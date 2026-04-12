import type { Ecosystem } from "@/domains/common/types/common.types"

export const formatEcosystem = (ecosystem: Ecosystem | string): string => {
  switch (ecosystem) {
    case "python":
      return "Python"
    case "npm":
      return "NPM"
    default:
      return ecosystem
  }
}
