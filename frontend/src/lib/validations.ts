import { z } from "zod"

export const loginSchema = z.object({
  email: z.string().email("Please enter a valid email address"),
  password: z.string().min(1, "Password is required"),
})

export const registerSchema = z
  .object({
    firstName: z.string().min(1, "First name is required"),
    lastName: z.string().min(1, "Last name is required"),
    email: z.string().email("Please enter a valid email address"),
    password: z.string().min(8, "Password must be at least 8 characters"),
    confirmPassword: z.string().min(1, "Please confirm your password"),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  })

export const packageSchema = z.object({
  name: z.string().min(1, "Package name is required").trim(),
  ecosystem: z.enum(["python", "npm"], { message: "Select an ecosystem" }),
})

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

export const invitationSchema = z.object({
  email: z.string().email("Please enter a valid email address"),
  roleId: z.number().positive("Please select a role"),
})

export const channelSchema = z.object({
  name: z.string().min(1, "Channel name is required"),
  type: z.enum(["email", "slack", "webhook"], {
    message: "Select a channel type",
  }),
  config: z.record(z.string(), z.string()).optional(),
})

export const profileSchema = z.object({
  firstName: z.string().min(1, "First name is required"),
  lastName: z.string().min(1, "Last name is required"),
})

export const passwordResetSchema = z.object({
  email: z.string().email("Please enter a valid email address"),
})

export const passwordChangeSchema = z
  .object({
    currentPassword: z.string().min(1, "Current password is required"),
    newPassword: z.string().min(8, "New password must be at least 8 characters"),
    confirmPassword: z.string().min(1, "Please confirm your new password"),
  })
  .refine((data) => data.newPassword === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  })

export const newPasswordSchema = z
  .object({
    password: z.string().min(8, "Password must be at least 8 characters"),
    confirmPassword: z.string().min(1, "Please confirm your password"),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords do not match",
    path: ["confirmPassword"],
  })

export const organizationSchema = z.object({
  name: z.string().min(1, "Organization name is required"),
  slug: z
    .string()
    .min(1, "Slug is required")
    .regex(
      /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
      "Slug must be lowercase letters, numbers, and hyphens"
    ),
  description: z.string().optional(),
})

// Type exports
export type LoginFormData = z.infer<typeof loginSchema>
export type RegisterFormData = z.infer<typeof registerSchema>
export type PackageFormData = z.infer<typeof packageSchema>
export type SettingsFormData = z.infer<typeof settingsSchema>
export type InvitationFormData = z.infer<typeof invitationSchema>
export type ChannelFormData = z.infer<typeof channelSchema>
export type ProfileFormData = z.infer<typeof profileSchema>
export type PasswordResetFormData = z.infer<typeof passwordResetSchema>
export type PasswordChangeFormData = z.infer<typeof passwordChangeSchema>
export type NewPasswordFormData = z.infer<typeof newPasswordSchema>
export type OrganizationFormData = z.infer<typeof organizationSchema>
