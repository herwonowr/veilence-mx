import "@testing-library/jest-dom/vitest"
import { server } from "@/__tests__/msw-server"

// ── MSW server lifecycle ──────────────────────────────────────
// Registered here (setupFiles) so server.listen() is guaranteed
// to fire before every test file and patch globalThis.fetch.
beforeAll(() => {
  server.listen({ onUnhandledRequest: "bypass" })
})
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

// Mock next/navigation
vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    forward: vi.fn(),
    refresh: vi.fn(),
    prefetch: vi.fn(),
  }),
  usePathname: () => "/",
  useSearchParams: () => new URLSearchParams(),
  redirect: vi.fn(),
  notFound: vi.fn(),
}))

// Mock next/font/google
vi.mock("next/font/google", () => ({
  Geist: () => ({ variable: "--font-geist-sans" }),
  Geist_Mono: () => ({ variable: "--font-geist-mono" }),
}))

// Provide a minimal localStorage mock (jsdom has one, but ensure it's clean)
beforeEach(() => {
  localStorage.clear()
})
