"use client"

import { Toaster as SonnerToaster } from "sonner"
import { useTheme } from "next-themes"

export const Toaster = () => {
  const { resolvedTheme } = useTheme()

  return (
    <SonnerToaster
      theme={resolvedTheme === "dark" ? "dark" : "light"}
      position="top-center"
    />
  )
}
