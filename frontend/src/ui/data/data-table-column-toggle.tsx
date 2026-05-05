"use client"

import type { Table } from "@tanstack/react-table"
import { Settings2, Check } from "lucide-react"
import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/ui"

interface DataTableColumnToggleProps<TData> {
  table: Table<TData>
}

export const DataTableColumnToggle = <TData,>({
  table,
}: DataTableColumnToggleProps<TData>) => {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button variant="outline" size="sm" className="h-8 gap-1.5">
            <Settings2 className="size-3.5" />
            Columns
          </Button>
        }
      />
      <DropdownMenuContent align="end">
        <DropdownMenuLabel>Toggle columns</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {table
          .getAllColumns()
          .filter((column) => column.getCanHide())
          .map((column) => (
            <DropdownMenuItem
              key={column.id}
              onClick={() => column.toggleVisibility(!column.getIsVisible())}
              className="capitalize gap-2"
            >
              <Check className={`size-3.5 ${column.getIsVisible() ? "opacity-100" : "opacity-0"}`} />
              {column.columnDef.meta?.title
                ?? (typeof column.columnDef.header === "string"
                  ? column.columnDef.header
                  : column.id.replace(/([A-Z])/g, " $1").trim())}
            </DropdownMenuItem>
          ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
