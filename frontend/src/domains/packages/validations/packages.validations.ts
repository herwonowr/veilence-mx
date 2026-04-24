import { z } from "zod"

export const packageSchema = z.object({
  name: z.string().min(1, "Package name is required").trim(),
  ecosystem: z.enum(["python", "npm"], { message: "Select an ecosystem" }),
})

export const bulkImportEntrySchema = z.object({
  name: z.string().min(1, "Package name is required"),
  ecosystem: z.enum(["python", "npm"], { message: "Invalid ecosystem" }),
})

export type PackageFormData = z.infer<typeof packageSchema>
export type BulkImportEntryData = z.infer<typeof bulkImportEntrySchema>
