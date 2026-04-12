export interface SettingsMap {
  [key: string]: string
}

export interface ReanalyzeAllResponse {
  message: string
  queued: number
  dead_retried: number
}
