"use client"

import { useState, useCallback } from "react"
import { packageSchema } from "@/domains/packages"
import {
  Button,
  Input,
  Field,
  FieldLabel,
  FieldError,
  Badge,
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/ui"
import { XIcon, PlusIcon } from "lucide-react"
import type { AddedPackage } from "@/features/onboarding/hooks/use-onboarding"

interface AddPackagesStepProps {
  addedPackages: AddedPackage[]
  isAdding: boolean
  onAddPackage: (name: string, ecosystem: "python" | "npm") => void
  onRemovePackage: (id: string) => void
  onNext: () => void
  onSkip: () => void
  onBack: () => void
}

export const AddPackagesStep = ({
  addedPackages,
  isAdding,
  onAddPackage,
  onRemovePackage,
  onNext,
  onSkip,
  onBack,
}: AddPackagesStepProps) => {
  const [name, setName] = useState("")
  const [ecosystem, setEcosystem] = useState<"python" | "npm">("python")
  const [error, setError] = useState("")

  const handleAdd = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault()
      const result = packageSchema.safeParse({ name, ecosystem })
      if (!result.success) {
        setError(result.error.issues[0]?.message ?? "Invalid input")
        return
      }
      setError("")
      onAddPackage(name.trim(), ecosystem)
      setName("")
    },
    [name, ecosystem, onAddPackage]
  )

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">Add Packages</h2>
        <p className="text-sm text-muted-foreground">
          Add specific packages you want to monitor. You can also skip this - top-N
          packages will be discovered automatically.
        </p>
      </div>

      <form onSubmit={handleAdd} className="flex items-end gap-2">
        <Field className="flex-1">
          <FieldLabel>Package Name</FieldLabel>
          <Input
            placeholder="e.g., requests, lodash"
            value={name}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setName(e.target.value)}
          />
          {error && <FieldError>{error}</FieldError>}
        </Field>
        <Field>
          <FieldLabel>Ecosystem</FieldLabel>
          <Select
            value={ecosystem}
            onValueChange={(val) => setEcosystem(val as "python" | "npm")}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="python">PyPI</SelectItem>
              <SelectItem value="npm">NPM</SelectItem>
            </SelectContent>
          </Select>
        </Field>
        <Button type="submit" size="sm" disabled={isAdding} className="mb-0.5">
          <PlusIcon className="size-4" />
          Add
        </Button>
      </form>

      {addedPackages.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-medium text-muted-foreground">
            Added ({addedPackages.length}):
          </p>
          <div className="flex flex-wrap gap-1.5">
            {addedPackages.map((pkg) => (
              <Badge key={pkg.id} variant="secondary" className="gap-1 pr-1">
                <span className="text-[10px] uppercase text-muted-foreground">
                  {pkg.ecosystem === "python" ? "pypi" : "npm"}
                </span>
                {pkg.name}
                <button
                  type="button"
                  onClick={() => onRemovePackage(pkg.id)}
                  className="ml-0.5 rounded-sm p-0.5 hover:bg-muted"
                >
                  <XIcon className="size-3" />
                </button>
              </Badge>
            ))}
          </div>
        </div>
      )}

      <div className="flex gap-2">
        <Button type="button" variant="outline" onClick={onBack} className="flex-1">
          Back
        </Button>
        <Button type="button" variant="ghost" onClick={onSkip}>
          Skip
        </Button>
        <Button
          type="button"
          onClick={onNext}
          disabled={addedPackages.length === 0}
          className="flex-1"
        >
          Continue
        </Button>
      </div>
    </div>
  )
}
