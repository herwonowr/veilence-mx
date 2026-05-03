"use client"

import { useState } from "react"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Button,
  Badge,
  Input,
  EmptyState,
} from "@/ui"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui"
import { Users, Search, ShieldCheck, UserX, UserCheck, Loader2 } from "lucide-react"
import { usePlatformUsers, useUpdatePlatformUser } from "@/features/platform-users"
import type { PlatformUserSummary } from "@/domains/platform-admin"

interface PasswordPromptState {
  userId: string
  action: "toggleAdmin" | "toggleActive"
  currentValue: boolean
}

export const PlatformUsersList = () => {
  const [search, setSearch] = useState("")
  const [debouncedSearch, setDebouncedSearch] = useState("")
  const [page, setPage] = useState(1)
  const [passwordPrompt, setPasswordPrompt] = useState<PasswordPromptState | null>(null)
  const [confirmPassword, setConfirmPassword] = useState("")

  const { data: usersRes, isLoading } = usePlatformUsers({
    page,
    pageSize: 20,
    search: debouncedSearch,
  })
  const updateMutation = useUpdatePlatformUser()

  const users = usersRes?.data?.users ?? []
  const total = usersRes?.data?.total ?? 0
  const totalPages = usersRes?.data?.totalPages ?? 1

  const handleSearchKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      setDebouncedSearch(search)
      setPage(1)
    }
  }

  const handleConfirmAction = () => {
    if (!passwordPrompt || !confirmPassword) return

    if (passwordPrompt.action === "toggleAdmin") {
      updateMutation.mutate(
        {
          id: passwordPrompt.userId,
          req: { isSuperAdmin: !passwordPrompt.currentValue },
          confirmPassword,
        },
        {
          onSuccess: () => {
            setPasswordPrompt(null)
            setConfirmPassword("")
          },
        }
      )
    } else {
      updateMutation.mutate(
        {
          id: passwordPrompt.userId,
          req: { isActive: !passwordPrompt.currentValue },
          confirmPassword,
        },
        {
          onSuccess: () => {
            setPasswordPrompt(null)
            setConfirmPassword("")
          },
        }
      )
    }
  }

  const authMethodLabel = (method: string) => {
    switch (method) {
      case "password":
        return "Password"
      case "google":
        return "Google"
      case "github":
        return "GitHub"
      case "saml":
        return "SAML"
      default:
        return method
    }
  }

  const promptDescription = passwordPrompt
    ? passwordPrompt.action === "toggleAdmin"
      ? passwordPrompt.currentValue
        ? "Enter your password to remove super admin privileges."
        : "Enter your password to grant super admin privileges."
      : passwordPrompt.currentValue
        ? "Enter your password to deactivate this user. All their sessions will be revoked."
        : "Enter your password to reactivate this user."
    : ""

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            Platform Users
          </CardTitle>
          <CardDescription>
            Manage all users on the platform. Total: {total}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search by email or name..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                onKeyDown={handleSearchKeyDown}
                className="pl-8"
              />
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setDebouncedSearch(search)
                setPage(1)
              }}
            >
              Search
            </Button>
          </div>

          {users.length === 0 ? (
            <EmptyState
              icon={<Users className="size-10" />}
              title="No users found"
              description="No users match your search criteria."
            />
          ) : (
            <div className="space-y-2">
              {users.map((user: PlatformUserSummary) => (
                <div
                  key={user.id}
                  className="flex items-center justify-between rounded-lg border p-3"
                >
                  <div>
                    <div className="flex items-center gap-2">
                      <span className="font-medium">
                        {user.firstName} {user.lastName}
                      </span>
                      <span className="text-sm text-muted-foreground">
                        {user.email}
                      </span>
                      {user.isSuperAdmin && (
                        <Badge variant="default" className="text-xs">
                          Super Admin
                        </Badge>
                      )}
                      {!user.isActive && (
                        <Badge variant="destructive" className="text-xs">
                          Deactivated
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground mt-1">
                      Auth: {authMethodLabel(user.authMethod)}
                      {" - "}
                      Joined {new Date(user.createdAt).toLocaleDateString()}
                      {user.lastLoginAt && (
                        <>
                          {" - "}
                          Last login{" "}
                          {new Date(user.lastLoginAt).toLocaleDateString()}
                        </>
                      )}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() =>
                        setPasswordPrompt({
                          userId: user.id,
                          action: "toggleAdmin",
                          currentValue: user.isSuperAdmin,
                        })
                      }
                      title={
                        user.isSuperAdmin
                          ? "Remove super admin"
                          : "Make super admin"
                      }
                    >
                      <ShieldCheck
                        className={`size-4 ${user.isSuperAdmin ? "text-primary" : "text-muted-foreground"}`}
                      />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() =>
                        setPasswordPrompt({
                          userId: user.id,
                          action: "toggleActive",
                          currentValue: user.isActive,
                        })
                      }
                      title={
                        user.isActive ? "Deactivate user" : "Reactivate user"
                      }
                    >
                      {user.isActive ? (
                        <UserX className="size-4 text-muted-foreground" />
                      ) : (
                        <UserCheck className="size-4 text-green-600" />
                      )}
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 pt-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                Previous
              </Button>
              <span className="text-sm text-muted-foreground">
                Page {page} of {totalPages}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                Next
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Password confirmation dialog */}
      <AlertDialog
        open={!!passwordPrompt}
        onOpenChange={(open) => {
          if (!open) {
            setPasswordPrompt(null)
            setConfirmPassword("")
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Confirm Password</AlertDialogTitle>
            <AlertDialogDescription>
              {promptDescription}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <Input
            type="password"
            placeholder="Your password"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            autoFocus
          />
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleConfirmAction}
              disabled={!confirmPassword || updateMutation.isPending}
            >
              {updateMutation.isPending && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Confirm
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
