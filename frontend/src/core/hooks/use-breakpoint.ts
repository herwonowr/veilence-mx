"use client"

import { useState, useEffect } from "react"

export type Breakpoint = "mobile" | "tablet" | "desktop" | "xl"

const BREAKPOINTS = {
  tablet: 768,
  desktop: 1024,
  xl: 1440,
} as const

const BREAKPOINT_ORDER: Breakpoint[] = ["mobile", "tablet", "desktop", "xl"]

const isAtLeast = (current: Breakpoint, minimum: Breakpoint): boolean =>
  BREAKPOINT_ORDER.indexOf(current) >= BREAKPOINT_ORDER.indexOf(minimum)

export const useBreakpoint = (): Breakpoint => {
  const [breakpoint, setBreakpoint] = useState<Breakpoint>("desktop")

  useEffect(() => {
    const update = () => {
      const w = window.innerWidth
      if (w < BREAKPOINTS.tablet) setBreakpoint("mobile")
      else if (w < BREAKPOINTS.desktop) setBreakpoint("tablet")
      else if (w < BREAKPOINTS.xl) setBreakpoint("desktop")
      else setBreakpoint("xl")
    }

    update()

    const mqlTablet = window.matchMedia(`(max-width: ${BREAKPOINTS.tablet - 1}px)`)
    const mqlDesktop = window.matchMedia(`(max-width: ${BREAKPOINTS.desktop - 1}px)`)
    const mqlXl = window.matchMedia(`(max-width: ${BREAKPOINTS.xl - 1}px)`)

    mqlTablet.addEventListener("change", update)
    mqlDesktop.addEventListener("change", update)
    mqlXl.addEventListener("change", update)

    return () => {
      mqlTablet.removeEventListener("change", update)
      mqlDesktop.removeEventListener("change", update)
      mqlXl.removeEventListener("change", update)
    }
  }, [])

  return breakpoint
}

export { isAtLeast }
