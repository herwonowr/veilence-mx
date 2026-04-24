"use client"

import { Dialog, DialogContent } from "@/ui"
import { WizardProgress } from "@/features/onboarding/ui/wizard-progress"
import { WelcomeStep } from "@/features/onboarding/ui/welcome-step"
import { CreateWorkspaceStep } from "@/features/onboarding/ui/create-workspace-step"
import { ConfigureSettingsStep } from "@/features/onboarding/ui/configure-settings-step"
import { AddPackagesStep } from "@/features/onboarding/ui/add-packages-step"
import { FirstDiscoveryStep } from "@/features/onboarding/ui/first-discovery-step"
import { CompleteStep } from "@/features/onboarding/ui/complete-step"
import { useOnboarding } from "@/features/onboarding/hooks/use-onboarding"

export const OnboardingWizard = () => {
  const {
    currentStep,
    totalSteps,
    workspaceForm,
    settingsForm,
    settingsErrors,
    addedPackages,
    discoveryTriggered,
    isCreatingWorkspace,
    isSavingSettings,
    isAddingPackage,
    isDiscovering,
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
  } = useOnboarding()

  return (
    <Dialog open modal>
      <DialogContent showCloseButton={false} className="sm:max-w-lg">
        <div className="space-y-6 max-h-[70vh] overflow-y-auto pr-1">
          <WizardProgress currentStep={currentStep} totalSteps={totalSteps} />

          {currentStep === 1 && <WelcomeStep onNext={goNext} />}

          {currentStep === 2 && (
            <CreateWorkspaceStep
              formData={workspaceForm}
              isCreating={isCreatingWorkspace}
              onUpdateField={updateWorkspaceField}
              onSubmit={handleCreateWorkspace}
              onBack={goBack}
            />
          )}

          {currentStep === 3 && (
            <ConfigureSettingsStep
              formData={settingsForm}
              errors={settingsErrors}
              isSaving={isSavingSettings}
              onUpdateField={updateSettingsField}
              onSubmit={handleSaveSettings}
              onSkip={handleSkipSettings}
              onBack={goBack}
            />
          )}

          {currentStep === 4 && (
            <AddPackagesStep
              addedPackages={addedPackages}
              isAdding={isAddingPackage}
              onAddPackage={handleAddPackage}
              onRemovePackage={handleRemovePackage}
              onNext={goNext}
              onSkip={handleSkipPackages}
              onBack={goBack}
            />
          )}

          {currentStep === 5 && (
            <FirstDiscoveryStep
              discoveryTriggered={discoveryTriggered}
              isDiscovering={isDiscovering}
              onTriggerDiscovery={handleTriggerDiscovery}
              onNext={goNext}
              onSkip={goNext}
              onBack={goBack}
            />
          )}

          {currentStep === 6 && <CompleteStep onComplete={handleComplete} />}
        </div>
      </DialogContent>
    </Dialog>
  )
}
