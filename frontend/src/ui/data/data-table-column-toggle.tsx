"use client"

import type { Table } from "@tanstack/react-table"
import { Settings2 } from "lucide-react"
import {
  Button,
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
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
          <Button variant="outline" size="sm" className="h-8 gap-1.5" />
        }
      >
        <Settings2 className="size-3.5" />
        Columns
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuLabel>Toggle columns</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {table
          .getAllColumns()
          .filter((column) => column.getCanHide())
          .map((column) => (
            <DropdownMenuCheckboxItem
              key={column.id}
              checked={column.getIsVisible()}
              onClick={() => column.toggleVisibility(!column.getIsVisible())}
              className="capitalize"
            >
              {typeof column.columnDef.header === "string"
                ? column.columnDef.header
                : column.id.replace(/([A-Z])/g, " $1").trim()}
            </DropdownMenuCheckboxItem>
          ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
