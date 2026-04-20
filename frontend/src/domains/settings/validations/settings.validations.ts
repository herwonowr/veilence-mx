import { z } from "zod"

export const settingsSchema = z.object({
  monitoring_interval: z.string().optional(),
  discovery_scan_depth: z.string().optional(),
  discovery_interval: z.string().optional(),
  discovery_auto_approve: z.enum(["true", "false"]).optional(),
  stale_auto_remove_months: z.string().optional(),
  package_count_warning_threshold: z.string().optional(),
  email_digest_enabled: z.enum(["true", "false"]).optional(),
  email_digest_frequency: z.enum(["daily", "weekly"]).optional(),
  email_digest_recipients: z.string().optional(),
  diff_size_limit: z.string().optional(),
})

export type SettingsFormData = z.infer<typeof settingsSchema>
