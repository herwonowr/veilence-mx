import { z } from "zod"

const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

const commaSeparatedEmails = z
  .string()
  .min(1, "At least one recipient email is required")
  .refine(
    (val) =>
      val
        .split(",")
        .map((e) => e.trim())
        .filter(Boolean)
        .every((e) => emailRegex.test(e)),
    { message: "All entries must be valid email addresses" }
  )

export const emailConfigSchema = z.object({
  host: z.string().min(1, "SMTP host is required"),
  port: z
    .string()
    .min(1, "SMTP port is required")
    .regex(/^\d+$/, "Port must be a number"),
  username: z.string().optional().default(""),
  password: z.string().optional().default(""),
  from: z
    .string()
    .min(1, "From address is required")
    .regex(emailRegex, "Must be a valid email address"),
  to: commaSeparatedEmails,
})

export const slackConfigSchema = z.object({
  webhookUrl: z
    .string()
    .min(1, "Webhook URL is required")
    .refine((val) => val.startsWith("https://hooks.slack.com/"), {
      message: "Must start with https://hooks.slack.com/",
    }),
})

export const webhookConfigSchema = z.object({
  url: z
    .string()
    .min(1, "Webhook URL is required")
    .refine((val) => val.startsWith("http://") || val.startsWith("https://"), {
      message: "Must start with http:// or https://",
    }),
  secret: z.string().optional().default(""),
})

export const channelSchema = z.object({
  name: z.string().min(1, "Channel name is required"),
  type: z.enum(["email", "slack", "webhook"], {
    message: "Select a channel type",
  }),
  config: z.record(z.string(), z.string()).optional(),
})

export const notificationRuleSchema = z.object({
  channelId: z.string().min(1, "Channel is required"),
  severity: z.enum(["critical", "high", "medium", "low"], {
    message: "Select a severity level",
  }),
})

export type ChannelFormData = z.infer<typeof channelSchema>
export type NotificationRuleFormData = z.infer<typeof notificationRuleSchema>
export type EmailConfigData = z.infer<typeof emailConfigSchema>
export type SlackConfigData = z.infer<typeof slackConfigSchema>
export type WebhookConfigData = z.infer<typeof webhookConfigSchema>
