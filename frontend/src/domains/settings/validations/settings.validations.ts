import { z } from "zod"

export const settingsSchema = z.object({
  python_poll_interval: z.string().optional(),
  npm_poll_interval: z.string().optional(),
  python_top_n: z.string().optional(),
  npm_top_n: z.string().optional(),
  version_depth_mode: z.enum(["latest", "custom"]).optional(),
  version_depth_count: z.string().optional(),
  diff_size_limit: z.string().optional(),
  email_digest_enabled: z.enum(["true", "false"]).optional(),
  email_digest_frequency: z.enum(["daily", "weekly"]).optional(),
  email_digest_recipients: z.string().optional(),
})

export type SettingsFormData = z.infer<typeof settingsSchema>
