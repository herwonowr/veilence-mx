import { z } from "zod"

export const invitationSchema = z.object({
  email: z.string().email("Please enter a valid email address"),
  roleId: z.number().positive("Please select a role"),
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

export type InvitationFormData = z.infer<typeof invitationSchema>
export type OrganizationFormData = z.infer<typeof organizationSchema>
