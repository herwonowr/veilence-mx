"use client"

import { useCallback, useEffect, useState } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useAuth } from "@/core/providers/auth-provider"
import { apiCreateOrg } from "@/domains/admin"
import type { Organization } from "@/domains/admin"
import { Button } from "@/ui/components/button"
import { Input } from "@/ui/components/input"
import { Field, FieldLabel, FieldDescription } from "@/ui/components/field"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/ui/components/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/ui/components/dialog"
import { Badge } from "@/ui/components/badge"
import { EmptyState } from "@/ui/feedback/empty-state"
import { Building2, Plus, Loader2, Users, Package } from "lucide-react"
import { Alert, AlertDescription } from "@/ui/components/alert"
import Link from "next/link"
import { useOrgMembers } from "@/features/admin/hooks/use-organizations"
import { useAdminPackages } from "@/features/admin/hooks/use-admin-packages"

export const OrganizationsListView = () => {
  const { organizations, refreshOrgs, setCurrentOrg } = useAuth()
  const router = useRouter()
  const searchParams = useSearchParams()
  const shouldCreateOrg = searchParams.get("create") === "true"

  const [dialogOpen, setDialogOpen] = useState(shouldCreateOrg)
  const [name, setName] = useState("")
  const [slug, setSlug] = useState("")
  const [description, setDescription] = useState("")
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState("")

  useEffect(() => {
    if (shouldCreateOrg) {
      setDialogOpen(true)
      // Clean up the URL so subsequent navigations to ?create=true trigger the effect again
      router.replace("/organizations", { scroll: false })
    }
  }, [shouldCreateOrg, router])

  const generateSlug = useCallback((value: string) => {
    return value
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/^-|-$/g, "")
  }, [])

  const handleNameChange = (value: string) => {
    setName(value)
    setSlug(generateSlug(value))
  }

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    setCreating(true)

    try {
      const { data } = await apiCreateOrg({ name, slug, description })
      setCurrentOrg(data)
      await refreshOrgs()
      setDialogOpen(false)
      setName("")
      setSlug("")
      setDescription("")
      router.push(`/organizations/${data.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create organization")
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Organizations</h1>
        <Dialog open={dialogOpen} onOpenChange={(open) => setDialogOpen(open)}>
          <DialogTrigger
            render={
              <Button>
                <Plus className="mr-2 size-4" />
                New Organization
              </Button>
            }
          />
          <DialogContent className="sm:max-w-md">
            <form onSubmit={handleCreate}>
              <DialogHeader>
                <DialogTitle>Create Organization</DialogTitle>
                <DialogDescription>
                  Create a new organization to manage packages and team members.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4 py-4">
                {error && (
                  <Alert variant="destructive">
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}
                <Field>
                  <FieldLabel htmlFor="org-name">Name</FieldLabel>
                  <Input
                    id="org-name"
                    placeholder="My Organization"
                    value={name}
                    onChange={(e) => handleNameChange(e.target.value)}
                    required
                  />
                </Field>
                <Field>
                  <FieldLabel htmlFor="org-slug">Slug</FieldLabel>
                  <Input
                    id="org-slug"
                    placeholder="my-organization"
                    value={slug}
                    onChange={(e) => setSlug(e.target.value)}
                    required
                  />
                  <FieldDescription>
                    URL-friendly identifier for your organization
                  </FieldDescription>
                </Field>
                <Field>
                  <FieldLabel htmlFor="org-description">Description</FieldLabel>
                  <Input
                    id="org-description"
                    placeholder="Optional description"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                  />
                </Field>
              </div>
              <DialogFooter>
                <Button type="submit" disabled={creating}>
                  {creating && (
                    <Loader2 className="mr-2 size-4 animate-spin" />
                  )}
                  Create
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {organizations.length === 0 ? (
        <Card>
          <CardContent>
            <EmptyState
              icon={<Building2 className="h-12 w-12" />}
              title="No organizations yet"
              description="Create your first organization to get started."
            >
              <Button onClick={() => setDialogOpen(true)}>
                <Plus className="mr-2 size-4" />
                Create Organization
              </Button>
            </EmptyState>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {organizations.map((org: Organization) => (
            <OrgCard key={org.id} org={org} />
          ))}
        </div>
      )}
    </div>
  )
}

const OrgCard = ({ org }: { org: Organization }) => {
  const { data: membersRes } = useOrgMembers(org.id)
  const { data: packagesRes } = useAdminPackages({ page: 1, limit: 1 })

  const memberCount = membersRes?.data?.length ?? null
  const packageCount = packagesRes?.meta?.total ?? null

  return (
    <Link href={`/organizations/${org.id}`}>
      <Card className="cursor-pointer transition-colors hover:bg-muted/50">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-base">{org.name}</CardTitle>
            <Badge variant={org.isActive ? "secondary" : "outline"}>
              {org.isActive ? "Active" : "Inactive"}
            </Badge>
          </div>
          <CardDescription className="font-mono text-xs">
            {org.slug}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground line-clamp-2">
            {org.description || "No description"}
          </p>
          <div className="flex items-center gap-4 mt-3">
            <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Users className="size-3.5" />
              {memberCount !== null ? memberCount : "—"} {memberCount === 1 ? "member" : "members"}
            </span>
            <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Package className="size-3.5" />
              {packageCount !== null ? packageCount : "—"} {packageCount === 1 ? "package" : "packages"}
            </span>
          </div>
          <p className="text-xs text-muted-foreground mt-2">
            Created {new Date(org.createdAt).toLocaleDateString()}
          </p>
        </CardContent>
      </Card>
    </Link>
  )
}

