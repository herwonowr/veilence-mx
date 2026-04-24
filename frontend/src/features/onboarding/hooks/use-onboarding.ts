"use client"

import { useCallback, useState } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { useRouter } from "next/navigation"
import { toast } from "sonner"
import { useAuth, sanitizeErrorMessage } from "@/core"
import { apiCreateWorkspace } from "@/domains/admin"
import { updateSettings, onboardingSettingsSchema } from "@/domains/settings"
import { createPackage, discoverPackages } from "@/domains/packages"
import { onboardingKeys } from "@/features/onboarding/hooks/use-onboarding-check"

export type OnboardingStep = 1 | 2 | 3 | 4 | 5 | 6

export interface WorkspaceFormData {
  name: string
  slug: string
  description: string
}

export interface SettingsFormData {
  discovery_scan_depth: string
  monitoring_interval: string
  discovery_interval: string
}

export interface AddedPackage {
  id: string
  name: string
  ecosystem: "python" | "npm"
}

export interface DiscoveryPreference {
  enabled: boolean
  ecosystem?: string
}

const TOTAL_STEPS = 6

const toSlug = (name: string): string =>
  name
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")

export const useOnboarding = () => {
  const router = useRouter()
  const queryClient = useQueryClient()
  const { setCurrentWorkspace, refreshWorkspaces } = useAuth()

  const [currentStep, setCurrentStep] = useState<OnboardingStep>(1)
  const [workspaceForm, setWorkspaceForm] = useState<WorkspaceFormData>({
    name: "",
    slug: "",
    description: "",
  })
  const [slugManuallyEdited, setSlugManuallyEdited] = useState(false)
  const [settingsErrors, setSettingsErrors] = useState<Record<string, string>>({})
  const [settingsSkipped, setSettingsSkipped] = useState(false)

  const [settingsForm, setSettingsForm] = useState<SettingsFormData>({
    discovery_scan_depth: "100",
    monitoring_interval: "1h",
    discovery_interval: "24h",
  })
  const [addedPackages, setAddedPackages] = useState<AddedPackage[]>([])
  const [localIdCounter, setLocalIdCounter] = useState(0)
  const [discoveryPreference, setDiscoveryPreference] = useState<DiscoveryPreference>({
    enabled: false,
  })

  const [isSubmitting, setIsSubmitting] = useState(false)

  const updateWorkspaceField = useCallback(
    (field: keyof WorkspaceFormData, value: string) => {
      setWorkspaceForm((prev) => {
        const next = { ...prev, [field]: value }
        if (field === "name" && !slugManuallyEdited) {
          next.slug = toSlug(value)
        }
        if (field === "slug") {
          setSlugManuallyEdited(true)
        }
        return next
      })
    },
    [slugManuallyEdited]
  )

  const updateSettingsField = useCallback(
    (field: keyof SettingsFormData, value: string) => {
      setSettingsForm((prev) => ({ ...prev, [field]: value }))
    },
    []
  )

  const goNext = useCallback(() => {
    setCurrentStep((prev) => Math.min(prev + 1, TOTAL_STEPS) as OnboardingStep)
  }, [])

  const goBack = useCallback(() => {
    setCurrentStep((prev) => Math.max(prev - 1, 1) as OnboardingStep)
  }, [])

  // Step 2: validate workspace form locally, then advance
  const handleCreateWorkspace = useCallback(() => {
    goNext()
  }, [goNext])

  // Step 3: validate settings locally, then advance
  const handleSaveSettings = useCallback(() => {
    const result = onboardingSettingsSchema.safeParse(settingsForm)
    if (!result.success) {
      const fieldErrors: Record<string, string> = {}
      for (const issue of result.error.issues) {
        const key = issue.path[0]
        if (typeof key === "string") {
          fieldErrors[key] = issue.message
        }
      }
      setSettingsErrors(fieldErrors)
      return
    }
    setSettingsErrors({})
    setSettingsSkipped(false)
    goNext()
  }, [settingsForm, goNext])

  const handleSkipSettings = useCallback(() => {
    setSettingsSkipped(true)
    setCurrentStep(4)
  }, [])

  // Step 4: add package to local list (no API call)
  const handleAddPackage = useCallback(
    (name: string, ecosystem: "python" | "npm") => {
      const isDuplicate = addedPackages.some(
        (p) => p.name === name && p.ecosystem === ecosystem
      )
      if (isDuplicate) {
        toast.error("This package has already been added")
        return
      }
      setLocalIdCounter((prev) => prev + 1)
      setAddedPackages((prev) => [
        ...prev,
        {
          id: `local-${Date.now()}-${localIdCounter + 1}`,
          name,
          ecosystem,
        },
      ])
    },
    [addedPackages, localIdCounter]
  )

  const handleRemovePackage = useCallback((id: string) => {
    setAddedPackages((prev) => prev.filter((p) => p.id !== id))
  }, [])

  const handleSkipPackages = useCallback(() => {
    setCurrentStep(5)
  }, [])

  // Step 5: record discovery preference locally
  const handleTriggerDiscovery = useCallback(
    (ecosystem?: string) => {
      setDiscoveryPreference({ enabled: true, ecosystem })
    },
    []
  )

  // Step 6: submit everything in sequence
  const handleComplete = useCallback(async () => {
    setIsSubmitting(true)
    try {
      // 1. Create workspace
      const wsResponse = await apiCreateWorkspace({
        name: workspaceForm.name,
        slug: workspaceForm.slug,
        description: workspaceForm.description || undefined,
      })
      if (!wsResponse.data) {
        throw new Error("Failed to create workspace - no data returned")
      }

      // 2. Set current workspace and refresh
      setCurrentWorkspace(wsResponse.data)
      await refreshWorkspaces()

      // 3. Save settings (if not skipped)
      if (!settingsSkipped) {
        const settings: Record<string, string> = { ...settingsForm }
        await updateSettings(settings)
      }

      // 4. Add packages (if any)
      for (const pkg of addedPackages) {
        await createPackage(pkg.name, pkg.ecosystem)
      }

      // 5. Trigger discovery (if user chose to)
      if (discoveryPreference.enabled) {
        await discoverPackages(discoveryPreference.ecosystem)
      }

      // 6. Invalidate queries and redirect
      queryClient.invalidateQueries({ queryKey: onboardingKeys.check })
      queryClient.invalidateQueries({ queryKey: ["packages"] })
      queryClient.invalidateQueries({ queryKey: ["settings"] })
      router.push("/")
    } catch (error) {
      toast.error(
        sanitizeErrorMessage(
          error instanceof Error ? error : new Error(String(error)),
          "Failed to complete onboarding"
        )
      )
    } finally {
      setIsSubmitting(false)
    }
  }, [
    workspaceForm,
    settingsForm,
    settingsSkipped,
    addedPackages,
    discoveryPreference,
    setCurrentWorkspace,
    refreshWorkspaces,
    queryClient,
    router,
  ])

  return {
    currentStep,
    totalSteps: TOTAL_STEPS,
    workspaceForm,
    settingsForm,
    settingsErrors,
    addedPackages,
    discoveryPreference,
    slugManuallyEdited,
    isSubmitting,

    // Actions
    goNext,
    goBack,
    updateWorkspaceField,
    updateSettingsField,
    handleCreateWorkspace,
    handleSaveSettings,
    handleSkipSettings,
    handleAddPackage,
    handleRemovePackage,
    handleSkipPackages,
    handleTriggerDiscovery,
    handleComplete,
  }
}
