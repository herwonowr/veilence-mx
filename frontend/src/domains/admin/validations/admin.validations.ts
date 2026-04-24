import { z } from "zod"

export const invitationSchema = z.object({
  email: z.email("Please enter a valid email address"),
  roleId: z.number().positive("Please select a role"),
})

export const workspaceSchema = z.object({
  name: z
    .string()
    .min(2, "Workspace name must be at least 2 characters")
    .max(100, "Workspace name must be at most 100 characters"),
  slug: z
    .string()
    .min(2, "Slug must be at least 2 characters")
    .max(50, "Slug must be at most 50 characters")
    .regex(
      /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
      "Slug must be lowercase letters, numbers, and hyphens"
    ),
  description: z.string().max(500, "Description must be at most 500 characters").optional(),
})

export const workspaceUpdateSchema = z.object({
  name: z
    .string()
    .min(2, "Workspace name must be at least 2 characters")
    .max(100, "Workspace name must be at most 100 characters"),
  description: z.string().max(500, "Description must be at most 500 characters").optional(),
})

export type InvitationFormData = z.infer<typeof invitationSchema>
export type WorkspaceFormData = z.infer<typeof workspaceSchema>
export type WorkspaceUpdateFormData = z.infer<typeof workspaceUpdateSchema>
