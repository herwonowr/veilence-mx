/**
 * Format version with "v" prefix, avoiding double "v" for Go modules.
 */
export const formatVersion = (version: string): string =>
  version.startsWith("v") ? version : `v${version}`
