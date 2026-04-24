import { z } from "zod"

export const alertNoteSchema = z.object({
  content: z.string().min(1, "Note content is required").max(5000, "Note too long"),
})

export type AlertNoteFormData = z.infer<typeof alertNoteSchema>
