import { z } from "zod"

const numericString = (label: string) =>
  z
    .string()
    .optional()
    .refine((val) => !val || /^\d+$/.test(val), {
      message: `${label} must be a number`,
    })

const durationString = (label: string) =>
  z
    .string()
    .optional()
    .refine((val) => !val || /^\d+[smhd]$/.test(val), {
      message: `${label} must be a duration (e.g., 30m, 1h, 24h)`,
    })

const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

const optionalCommaSeparatedEmails = z
  .string()
  .optional()
  .refine(
    (val) =>
      !val ||
      val
        .split(",")
        .map((e) => e.trim())
        .filter(Boolean)
        .every((e) => emailRegex.test(e)),
    { message: "All entries must be valid email addresses" }
  )

export const settingsSchema = z.object({
  monitoring_interval: durationString("Monitoring interval"),
  discovery_scan_depth: numericString("Discovery scan depth").refine(
    (val) => !val || Number(val) <= 1000,
    { message: "Must be at most 1,000" }
  ),
  discovery_interval: durationString("Discovery interval"),
  discovery_auto_approve: z.enum(["true", "false"]).optional(),
  stale_auto_remove_months: numericString("Stale auto-remove months"),
  package_count_warning_threshold: numericString("Package count warning threshold"),
  email_digest_enabled: z.enum(["true", "false"]).optional(),
  email_digest_frequency: z.enum(["daily", "weekly"]).optional(),
  email_digest_recipients: optionalCommaSeparatedEmails,
})

export type SettingsFormData = z.infer<typeof settingsSchema>

export const onboardingSettingsSchema = z.object({
  discovery_scan_depth: z
    .string()
    .min(1, "Discovery scan depth is required")
    .refine((val) => /^\d+$/.test(val), { message: "Must be a number" })
    .refine((val) => Number(val) >= 1 && Number(val) <= 1000, {
      message: "Must be between 1 and 1,000",
    }),
  monitoring_interval: z
    .string()
    .min(1, "Monitoring interval is required")
    .refine((val) => /^\d+[smhd]$/.test(val), {
      message: "Must be a duration (e.g., 30m, 1h, 6h)",
    }),
  discovery_interval: z
    .string()
    .min(1, "Discovery interval is required")
    .refine((val) => /^\d+[smhd]$/.test(val), {
      message: "Must be a duration (e.g., 12h, 24h, 7d)",
    }),
})

export type OnboardingSettingsFormData = z.infer<typeof onboardingSettingsSchema>
