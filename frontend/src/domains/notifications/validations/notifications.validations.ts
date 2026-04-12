import { z } from "zod"

export const channelSchema = z.object({
  name: z.string().min(1, "Channel name is required"),
  type: z.enum(["email", "slack", "webhook"], {
    message: "Select a channel type",
  }),
  config: z.record(z.string(), z.string()).optional(),
})

export type ChannelFormData = z.infer<typeof channelSchema>
