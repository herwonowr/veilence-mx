"use client"

import { useCallback, useState } from "react"
import { useMutation, useQueryClient } from "@tanstack/react-query"
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

  const [settingsForm, setSettingsForm] = useState<SettingsFormData>({
    discovery_scan_depth: "100",
    monitoring_interval: "1800",
    discovery_interval: "86400",
  })
  const [addedPackages, setAddedPackages] = useState<AddedPackage[]>([])
  const [discoveryTriggered, setDiscoveryTriggered] = useState(false)

  // Workspace creation mutation
  const createWorkspaceMutation = useMutation({
    mutationFn: () =>
      apiCreateWorkspace({
        name: workspaceForm.name,
        slug: workspaceForm.slug,
        description: workspaceForm.description || undefined,
      }),
    onSuccess: async (response) => {
      if (response.data) {
        setCurrentWorkspace(response.data)
        await refreshWorkspaces()
        setCurrentStep(3)
      }
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to create workspace"))
    },
  })

  // Settings update mutation
  const updateSettingsMutation = useMutation({
    mutationFn: () => {
      const settings: Record<string, string> = { ...settingsForm }
      return updateSettings(settings)
    },
    onSuccess: () => {
      setCurrentStep(4)
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to save settings"))
    },
  })

  // Add package mutation
  const addPackageMutation = useMutation({
    mutationFn: ({ name, ecosystem }: { name: string; ecosystem: "python" | "npm" }) =>
      createPackage(name, ecosystem),
    onSuccess: (response, variables) => {
      if (response.data) {
        setAddedPackages((prev) => [
          ...prev,
          {
            id: response.data?.id ?? "",
            name: variables.name,
            ecosystem: variables.ecosystem,
          },
        ])
      }
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to add package"))
    },
  })

  // Discovery mutation
  const discoverMutation = useMutation({
    mutationFn: (ecosystem?: string) => discoverPackages(ecosystem),
    onSuccess: () => {
      setDiscoveryTriggered(true)
      toast.success("Discovery started - results will appear in your dashboard")
    },
    onError: (error: Error) => {
      toast.error(sanitizeErrorMessage(error, "Failed to trigger discovery"))
    },
  })

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

  const handleCreateWorkspace = useCallback(() => {
    createWorkspaceMutation.mutate()
  }, [createWorkspaceMutation])

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
    updateSettingsMutation.mutate()
  }, [updateSettingsMutation, settingsForm])

  const handleSkipSettings = useCallback(() => {
    setCurrentStep(4)
  }, [])

  const handleAddPackage = useCallback(
    (name: string, ecosystem: "python" | "npm") => {
      const isDuplicate = addedPackages.some(
        (p) => p.name === name && p.ecosystem === ecosystem
      )
      if (isDuplicate) {
        toast.error("This package has already been added")
        return
      }
      addPackageMutation.mutate({ name, ecosystem })
    },
    [addPackageMutation, addedPackages]
  )

  const handleRemovePackage = useCallback((id: string) => {
    setAddedPackages((prev) => prev.filter((p) => p.id !== id))
  }, [])

  const handleSkipPackages = useCallback(() => {
    setCurrentStep(5)
  }, [])

  const handleTriggerDiscovery = useCallback(
    (ecosystem?: string) => {
      discoverMutation.mutate(ecosystem)
    },
    [discoverMutation]
  )

  const handleComplete = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: onboardingKeys.check })
    queryClient.invalidateQueries({ queryKey: ["packages"] })
    queryClient.invalidateQueries({ queryKey: ["settings"] })
    router.push("/")
  }, [queryClient, router])

  return {
    currentStep,
    totalSteps: TOTAL_STEPS,
    workspaceForm,
    settingsForm,
    settingsErrors,
    addedPackages,
    discoveryTriggered,
    slugManuallyEdited,

    // Mutations
    isCreatingWorkspace: createWorkspaceMutation.isPending,
    isSavingSettings: updateSettingsMutation.isPending,
    isAddingPackage: addPackageMutation.isPending,
    isDiscovering: discoverMutation.isPending,

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
