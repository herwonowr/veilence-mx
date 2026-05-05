import { NextRequest, NextResponse } from "next/server"
import { ROUTES, PUBLIC_PATHS, AUTH_PAGE_PATHS, SUPER_ADMIN_PATHS } from "@/core/routes"

/**
 * SSR route protection proxy.
 *
 * Checks for the `vmx_authenticated` UX-hint cookie (set by the client-side
 * auth provider) and redirects unauthenticated users away from protected
 * routes before any page HTML is sent, eliminating content flash.
 *
 * This is NOT a security boundary - the backend JWT check is authoritative.
 */

const AUTH_COOKIE = "vmx_authenticated"
const SUPER_ADMIN_COOKIE = "vmx_super_admin"

const isPublicPath = (pathname: string): boolean =>
  PUBLIC_PATHS.some((p) => pathname === p || pathname.startsWith(`${p}/`))

const isAuthPagePath = (pathname: string): boolean =>
  AUTH_PAGE_PATHS.some((p) => pathname === p || pathname.startsWith(`${p}/`))

const isSuperAdminPath = (pathname: string): boolean =>
  SUPER_ADMIN_PATHS.some((p) => pathname === p || pathname.startsWith(`${p}/`))

export const proxy = (request: NextRequest): NextResponse => {
  const { pathname } = request.nextUrl
  const hasAuthCookie = request.cookies.has(AUTH_COOKIE)
  const isSuperAdmin = request.cookies.get(SUPER_ADMIN_COOKIE)?.value === "1"

  // Authenticated user hitting an auth page - redirect to dashboard
  if (hasAuthCookie && isAuthPagePath(pathname)) {
    return NextResponse.redirect(new URL(ROUTES.DASHBOARD, request.url))
  }

  // Unauthenticated user hitting a protected page - redirect to login
  if (!hasAuthCookie && !isPublicPath(pathname)) {
    const loginUrl = new URL(ROUTES.LOGIN, request.url)
    loginUrl.searchParams.set("redirect", pathname)
    // Preserve SSO error params so login page can display them
    const error = request.nextUrl.searchParams.get("error")
    const message = request.nextUrl.searchParams.get("message")
    if (error) loginUrl.searchParams.set("error", error)
    if (message) loginUrl.searchParams.set("message", message)
    return NextResponse.redirect(loginUrl)
  }

  // Non-super-admin user hitting a super-admin page - redirect to dashboard
  if (hasAuthCookie && isSuperAdminPath(pathname) && !isSuperAdmin) {
    return NextResponse.redirect(new URL(ROUTES.DASHBOARD, request.url))
  }

  return NextResponse.next()
}

export const config = {
  matcher: [
    /*
     * Match all paths except:
     * - _next/static, _next/image (Next.js internals)
     * - favicon.ico, public assets
     * - API routes
     */
    "/((?!_next/static|_next/image|favicon\\.ico|api/).*)",
  ],
}
