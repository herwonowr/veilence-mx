"use client"

import { Suspense, useEffect, useState, useMemo, useCallback } from "react"
import {
  useDebouncedValue,
  useSortParams,
  useFilterParams,
  useResponsiveColumns,
  type ColumnBreakpoints,
} from "@/core"
import {
  Card,
  CardContent,
  CardHeader,
  Badge,
  Button,
  Input,
  SearchInput,
  Label,
  TableSkeleton,
  TableError,
  TableEmptyState,
  FilterChips,
  DataTablePagination,
  SortableHeader,
  type SkeletonColumn,
  type ActiveFilter,
} from "@/ui"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
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
import { Users, ShieldCheck, UserX, UserCheck, Loader2 } from "lucide-react"
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
  type PaginationState,
} from "@tanstack/react-table"
import "@/ui/data/table.types"
import { usePlatformUsers, useUpdatePlatformUser } from "@/features/platform-users/hooks/use-platform-users"
import type { PlatformUserSummary } from "@/domains/platform-admin"

interface PasswordPromptState {
  userId: string
  action: "toggleAdmin" | "toggleActive"
  currentValue: boolean
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

export const PlatformUsersList = () => (
  <Suspense>
    <PlatformUsersContent />
  </Suspense>
)

const PlatformUsersContent = () => {
  const [search, setSearch] = useState("")
  const debouncedSearch = useDebouncedValue(search, 300)
  const [statusFilter, setStatusFilter] = useState("")
  const [roleFilter, setRoleFilter] = useState("")
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 20,
  })
  const [sorting, setSorting] = useSortParams()

  const [passwordPrompt, setPasswordPrompt] = useState<PasswordPromptState | null>(null)
  const [confirmPassword, setConfirmPassword] = useState("")

  useFilterParams(
    useMemo(() => ({
      status: statusFilter,
      role: roleFilter,
    }), [statusFilter, roleFilter]),
  )

  const columnBreakpoints: ColumnBreakpoints = useMemo(() => ({
    authMethod: "desktop",
    createdAt: "desktop",
  }), [])
  const columnVisibility = useResponsiveColumns(columnBreakpoints)

  const sort = sorting[0]
  const { data: usersRes, isLoading, isFetching, isError, refetch } = usePlatformUsers({
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
    search: debouncedSearch || undefined,
    sortBy: sort?.id,
    sortDir: sort ? (sort.desc ? "desc" : "asc") : undefined,
    status: statusFilter || undefined,
    role: roleFilter || undefined,
  })

  const users = usersRes?.data?.users ?? []
  const total = usersRes?.data?.total ?? 0

  const updateMutation = useUpdatePlatformUser()

  useEffect(() => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [statusFilter, roleFilter, debouncedSearch, sorting])

  const hasActiveFilters = !!(search || statusFilter || roleFilter)

  const clearAllFilters = () => {
    setSearch("")
    setStatusFilter("")
    setRoleFilter("")
    setSorting([])
  }

  const activeFilters: ActiveFilter[] = [
    ...(statusFilter
      ? [{ label: "Status", value: statusFilter === "active" ? "Active" : "Inactive", onRemove: () => setStatusFilter("") }]
      : []),
    ...(roleFilter
      ? [{ label: "Role", value: roleFilter === "super_admin" ? "Super Admin" : "User", onRemove: () => setRoleFilter("") }]
      : []),
    ...(search
      ? [{ label: "Search", value: search, onRemove: () => setSearch("") }]
      : []),
  ]

  const handleConfirmAction = useCallback(() => {
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
  }, [passwordPrompt, confirmPassword, updateMutation])

  const promptDescription = passwordPrompt
    ? passwordPrompt.action === "toggleAdmin"
      ? passwordPrompt.currentValue
        ? "Enter your password to remove super admin privileges."
        : "Enter your password to grant super admin privileges."
      : passwordPrompt.currentValue
        ? "Enter your password to deactivate this user. All their sessions will be revoked."
        : "Enter your password to reactivate this user."
    : ""

  const skeletonColumns: SkeletonColumn[] = [
    { width: "w-32", header: "Name" },
    { width: "w-40", header: "Email" },
    { width: "w-16", header: "Status" },
    { width: "w-20", header: "Role" },
    { width: "w-20", header: "Auth Provider" },
    { width: "w-24", header: "Created" },
    { width: "w-20", header: "" },
  ]

  const columns = useMemo<ColumnDef<PlatformUserSummary>[]>(
    () => [
      {
        accessorKey: "firstName",
        header: ({ column }) => <SortableHeader column={column} title="Name" />,
        cell: ({ row }) => (
          <span className="font-medium">
            {row.original.firstName} {row.original.lastName}
          </span>
        ),
      },
      {
        accessorKey: "email",
        header: ({ column }) => <SortableHeader column={column} title="Email" />,
        cell: ({ row }) => (
          <span className="text-muted-foreground">{row.original.email}</span>
        ),
      },
      {
        accessorKey: "isActive",
        header: ({ column }) => <SortableHeader column={column} title="Status" />,
        cell: ({ row }) => (
          row.original.isActive ? (
            <Badge variant="outline">Active</Badge>
          ) : (
            <Badge variant="destructive">Inactive</Badge>
          )
        ),
      },
      {
        accessorKey: "isSuperAdmin",
        header: ({ column }) => <SortableHeader column={column} title="Role" />,
        cell: ({ row }) => (
          row.original.isSuperAdmin ? (
            <Badge variant="default">Super Admin</Badge>
          ) : (
            <span className="text-sm text-muted-foreground">User</span>
          )
        ),
      },
      {
        accessorKey: "authMethod",
        header: ({ column }) => <SortableHeader column={column} title="Auth Provider" />,
        cell: ({ row }) => (
          <span className="text-sm">{authMethodLabel(row.original.authMethod)}</span>
        ),
      },
      {
        accessorKey: "createdAt",
        header: ({ column }) => <SortableHeader column={column} title="Created" />,
        cell: ({ row }) => (
          <span className="text-sm whitespace-nowrap">
            {new Date(row.original.createdAt).toLocaleDateString()}
          </span>
        ),
      },
      {
        id: "actions",
        header: () => <span className="sr-only">Actions</span>,
        enableSorting: false,
        meta: { headerClassName: "w-[1%] whitespace-nowrap text-right", cellClassName: "text-right" },
        cell: ({ row }) => (
          <div className="flex items-center justify-end gap-1">
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() =>
                setPasswordPrompt({
                  userId: row.original.id,
                  action: "toggleAdmin",
                  currentValue: row.original.isSuperAdmin,
                })
              }
              title={
                row.original.isSuperAdmin
                  ? "Remove super admin"
                  : "Make super admin"
              }
            >
              <ShieldCheck
                className={`size-4 ${row.original.isSuperAdmin ? "text-primary" : "text-muted-foreground"}`}
              />
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() =>
                setPasswordPrompt({
                  userId: row.original.id,
                  action: "toggleActive",
                  currentValue: row.original.isActive,
                })
              }
              title={
                row.original.isActive ? "Deactivate user" : "Reactivate user"
              }
            >
              {row.original.isActive ? (
                <UserX className="size-4 text-muted-foreground" />
              ) : (
                <UserCheck className="size-4 text-green-600" />
              )}
            </Button>
          </div>
        ),
      },
    ],
    []
  )

  const pageCount = Math.max(1, Math.ceil(total / pagination.pageSize))

  // eslint-disable-next-line react-hooks/incompatible-library -- TanStack Table API is intentionally non-memoizable
  const table = useReactTable({
    data: users,
    columns,
    pageCount,
    state: { pagination, sorting, columnVisibility },
    onPaginationChange: setPagination,
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
    manualSorting: true,
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <h1 className="text-3xl font-bold">Platform Users</h1>
      </div>

      <Card>
        <CardHeader>
          <div className="space-y-3">
            <div className="flex flex-wrap items-end gap-4">
              <SearchInput
                value={search}
                onChange={setSearch}
                onClear={() => setSearch("")}
                isLoading={isFetching && !!debouncedSearch}
                placeholder="Search by name or email..."
                aria-label="Search users"
              />
              <div className="space-y-1">
                <Label htmlFor="users-status-filter" className="text-xs text-muted-foreground">
                  Status
                </Label>
                <Select
                  value={statusFilter || "all"}
                  onValueChange={(v) => setStatusFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="users-status-filter" className="w-40">
                    <SelectValue>{statusFilter ? (statusFilter === "active" ? "Active" : "Inactive") : "All Statuses"}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Statuses</SelectItem>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="inactive">Inactive</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1">
                <Label htmlFor="users-role-filter" className="text-xs text-muted-foreground">
                  Role
                </Label>
                <Select
                  value={roleFilter || "all"}
                  onValueChange={(v) => setRoleFilter(v === "all" ? "" : (v ?? ""))}
                >
                  <SelectTrigger id="users-role-filter" className="w-40">
                    <SelectValue>{roleFilter ? (roleFilter === "super_admin" ? "Super Admin" : "User") : "All Roles"}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Roles</SelectItem>
                    <SelectItem value="super_admin">Super Admin</SelectItem>
                    <SelectItem value="user">User</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            {hasActiveFilters && (
              <FilterChips filters={activeFilters} onClearAll={clearAllFilters} />
            )}
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            {isLoading ? (
              <TableSkeleton columns={skeletonColumns} rows={5} />
            ) : isError ? (
              <TableError colSpan={columns.length} onRetry={() => refetch()} />
            ) : (
              <Table>
                <TableHeader>
                  {table.getHeaderGroups().map((headerGroup) => (
                    <TableRow key={headerGroup.id}>
                      {headerGroup.headers.map((header) => {
                        const sorted = header.column.getIsSorted()
                        return (
                          <TableHead
                            key={header.id}
                            className={header.column.columnDef.meta?.headerClassName}
                            aria-sort={sorted === "asc" ? "ascending" : sorted === "desc" ? "descending" : undefined}
                          >
                            {header.isPlaceholder
                              ? null
                              : flexRender(header.column.columnDef.header, header.getContext())}
                          </TableHead>
                        )
                      })}
                    </TableRow>
                  ))}
                </TableHeader>
                <TableBody>
                  {table.getRowModel().rows.length ? (
                    table.getRowModel().rows.map((row) => (
                      <TableRow key={row.id}>
                        {row.getVisibleCells().map((cell) => (
                          <TableCell key={cell.id} className={cell.column.columnDef.meta?.cellClassName}>
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))
                  ) : (
                    hasActiveFilters ? (
                      <TableEmptyState
                        colSpan={columns.length}
                        icon={<Users className="h-8 w-8" />}
                        title="No users match your filters."
                      >
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={clearAllFilters}
                        >
                          Clear Filters
                        </Button>
                      </TableEmptyState>
                    ) : (
                      <TableEmptyState
                        colSpan={columns.length}
                        icon={<Users className="h-8 w-8" />}
                        title="No users found"
                        description="No users have been registered yet."
                      />
                    )
                  )}
                </TableBody>
              </Table>
            )}
          </div>

          <DataTablePagination table={table} total={total} />
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
