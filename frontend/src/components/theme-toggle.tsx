"use client"

import { useTheme } from "next-themes"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Moon, Sun, Monitor, Check } from "lucide-react"
import { useSyncExternalStore } from "react"

const subscribe = () => () => {}

const themeOptions = [
  { value: "light", label: "Light", icon: Sun },
  { value: "dark", label: "Dark", icon: Moon },
  { value: "system", label: "System", icon: Monitor },
] as const

export function ThemeToggle() {
  const { theme, setTheme } = useTheme()
  const mounted = useSyncExternalStore(subscribe, () => true, () => false)

  // During SSR / before hydration, render a non-disabled placeholder that
  // matches the mounted layout so there is no visual "disabled flash".
  if (!mounted) {
    return (
      <Button
        variant="ghost"
        size="icon"
        className="h-8 w-8"
        aria-label="Toggle theme"
      >
        <Sun className="h-4 w-4" />
        <span className="sr-only">Toggle theme</span>
      </Button>
    )
  }

  const ActiveIcon =
    theme === "dark" ? Moon : theme === "light" ? Sun : Monitor

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            aria-label={`Toggle theme (current: ${theme})`}
          />
        }
      >
        <ActiveIcon className="h-4 w-4" />
        <span className="sr-only">Toggle theme</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {themeOptions.map(({ value, label, icon: Icon }) => (
          <DropdownMenuItem
            key={value}
            onClick={() => setTheme(value)}
          >
            <Icon className="mr-2 h-4 w-4" />
            {label}
            {theme === value && (
              <Check className="ml-auto h-4 w-4 text-muted-foreground" />
            )}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
