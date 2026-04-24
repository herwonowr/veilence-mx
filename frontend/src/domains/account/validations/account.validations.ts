import { z } from "zod"

export const apiKeySchema = z.object({
  name: z.string().min(1, "Key name is required").max(100, "Key name too long"),
  role: z.enum(["owner", "admin", "member", "viewer"], {
    message: "Select a valid role",
  }),
  expiresAt: z.string().optional(),
})

export type ApiKeyFormData = z.infer<typeof apiKeySchema>
