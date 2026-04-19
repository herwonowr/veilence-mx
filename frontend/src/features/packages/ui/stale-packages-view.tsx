"use client"

import { useState, useMemo } from "react"
import Link from "next/link"
import { Card, CardContent, CardHeader, CardTitle } from "@/ui/components/card"
import { Badge } from "@/ui/components/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/ui/components/table"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/ui/components/select"
import { TableSkeleton, type SkeletonColumn } from "@/ui/feedback/table-skeleton"
import { TableError } from "@/ui/feedback/table-error"
import { TableEmptyState } from "@/ui/feedback/empty-state"
import { formatEcosystem } from "@/domains/common"
import { formatPopularity, popularityLabel } from "@/domains/packages"
import type { StalePackage } from "@/domains/packages"
import { ArrowLeft, Clock } from "lucide-react"
import { Label } from "@/ui/components/label"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/ui/components/tooltip"
import {
  useReactTable,
  getCoreRowModel,
  flexRender,
  type ColumnDef,
} from "@tanstack/react-table"
import { useStalePackages } from "@/features/packages/hooks/use-packages"

const THRESHOLD_OPTIONS = [1, 3, 6, 12, 18, 24]

export const StalePackagesView = () => {
  const [months, setMonths] = useState(6)

  const {
    data: staleRes,
    isLoading,
    isError,
    refetch,
  } = useStalePackages(months)

  const stalePackages = staleRes?.data ?? []

  const skeletonColumns: SkeletonColumn[] = [
    { width: "w-32", header: "Name" },
    { width: "w-16", header: "Ecosystem" },
    { width: "w-24", header: "Last Release" },
    { width: "w-16", header: "Days Since" },
    { width: "w-20", header: "Popularity" },
  ]

  const columns = useMemo<ColumnDef<StalePackage>[]>(
    () => [
      {
        accessorKey: "name",
        header: "Name",
        cell: ({ row }) => (
          <Link
            href={`/packages/${row.original.id}`}
            className="font-medium hover:underline"
          >
            {row.original.name}
          </Link>
        ),
      },
      {
        accessorKey: "ecosystem",
        header: "Ecosystem",
        cell: ({ row }) => (
          <Badge variant="outline">{formatEcosystem(row.original.ecosystem)}</Badge>
        ),
      },
      {
        id: "lastRelease",
        header: "Last Release",
        cell: ({ row }) =>
          row.original.lastReleaseAt
            ? new Date(row.original.lastReleaseAt).toLocaleDateString()
            : "Never",
      },
      {
        accessorKey: "daysSinceLastRelease",
        header: "Days Since",
        cell: ({ row }) => {
          const days = row.original.daysSinceLastRelease
          return (
            <span className={days > 365 ? "text-destructive font-medium" : ""}>
              {days}
            </span>
          )
        },
      },
      {
        id: "popularity",
        header: () => (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger render={<span>Popularity</span>} />
              <TooltipContent>Downloads/mo (Python) · Score (NPM)</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        ),
        cell: ({ row }) => {
          const pkg = row.original
          return (
            <span className="text-sm tabular-nums">
              {formatPopularity(pkg.ecosystem, pkg.downloadCount)}
            </span>
          )
        },
      },
    ],
    []
  )

  const table = useReactTable({
    data: stalePackages,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  return (
    <div className="space-y-6">
      <div>
        <Link
          href="/packages"
          className="text-sm text-muted-foreground hover:text-foreground flex items-center gap-1 mb-2"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Packages
        </Link>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <h1 className="text-3xl font-bold">Stale Packages</h1>
          <div className="flex items-center gap-2">
            <Label htmlFor="stale-threshold" className="text-sm text-muted-foreground whitespace-nowrap">
              No releases in
            </Label>
            <Select
              value={String(months)}
              onValueChange={(v) => { if (v) setMonths(parseInt(v, 10)) }}
            >
              <SelectTrigger id="stale-threshold" className="w-32">
                <SelectValue>{months} month{months !== 1 ? "s" : ""}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {THRESHOLD_OPTIONS.map((m) => (
                  <SelectItem key={m} value={String(m)}>
                    {m} month{m !== 1 ? "s" : ""}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <Clock className="h-4 w-4" />
            {stalePackages.length} package{stalePackages.length !== 1 ? "s" : ""} with no release in {months}+ months
          </CardTitle>
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
                      {headerGroup.headers.map((header) => (
                        <TableHead key={header.id}>
                          {header.isPlaceholder
                            ? null
                            : flexRender(header.column.columnDef.header, header.getContext())}
                        </TableHead>
                      ))}
                    </TableRow>
                  ))}
                </TableHeader>
                <TableBody>
                  {table.getRowModel().rows.length ? (
                    table.getRowModel().rows.map((row) => (
                      <TableRow key={row.id}>
                        {row.getVisibleCells().map((cell) => (
                          <TableCell key={cell.id}>
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))
                  ) : (
                    <TableEmptyState
                      colSpan={columns.length}
                      icon={<Clock className="h-8 w-8" />}
                      title="No stale packages."
                      description={`All monitored packages have had a release within the last ${months} months.`}
                    />
                  )}
                </TableBody>
              </Table>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
