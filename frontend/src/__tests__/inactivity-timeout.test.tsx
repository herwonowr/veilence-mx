/**
 * SEC-S4-10: Tests for inactivity timeout behavior in AuthProvider.
 */

import { renderHook, act, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { AuthProvider, useAuth } from "@/core/providers/auth-provider"
import * as apiClient from "@/core/http"

// Mock sonner toast
const mockToastWarning = vi.fn(() => "toast-id-1")
const mockToastDismiss = vi.fn()

vi.mock("sonner", () => ({
  toast: {
    warning: (...args: unknown[]) => mockToastWarning(...args),
    dismiss: (...args: unknown[]) => mockToastDismiss(...args),
  },
}))

// Mock the API client module
vi.mock("@/lib/api-client", async (importOriginal) => {
  const actual = await importOriginal<typeof apiClient>()
  return {
    ...actual,
    apiLogin: vi.fn(),
    apiRegister: vi.fn(),
    apiLogout: vi.fn(),
    apiGetMe: vi.fn(),
    apiGetWorkspaces: vi.fn(),
    getStoredAccessToken: vi.fn(() => null),
    getStoredRefreshToken: vi.fn(() => null),
    storeTokens: vi.fn(),
    clearTokens: vi.fn(),
    getStoredWorkspaceId: vi.fn(() => null),
    storeWorkspaceId: vi.fn(),
    clearWorkspaceId: vi.fn(),
  }
})

const mockUser = {
  id: 1,
  email: "test@example.com",
  firstName: "Test",
  lastName: "User",
  isActive: true,
  emailVerified: true,
  lastLoginAt: null,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const mockWorkspace = {
  id: 1,
  name: "Test Workspace",
  slug: "test-workspace",
  description: "Test workspace",
  ownerId: 1,
  isActive: true,
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
}

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return ({ children }: { children: React.ReactNode }) => {
    return (
      <QueryClientProvider client={queryClient}>
        <AuthProvider>{children}</AuthProvider>
      </QueryClientProvider>
    )
  }
}

describe("Inactivity timeout", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    mockToastWarning.mockClear()
    mockToastDismiss.mockClear()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("sets up event listeners when authenticated", async () => {
    const addSpy = vi.spyOn(window, "addEventListener")

    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue("token")
    vi.mocked(apiClient.apiGetMe).mockResolvedValue({
      data: mockUser,
      error: null,
    })
    vi.mocked(apiClient.apiGetWorkspaces).mockResolvedValue({
      data: [mockWorkspace],
      error: null,
    })
    vi.mocked(apiClient.getStoredWorkspaceId).mockReturnValue(null)

    renderHook(() => useAuth(), { wrapper: createWrapper() })

    await waitFor(() => {
      expect(addSpy).toHaveBeenCalledWith(
        "mousemove",
        expect.any(Function),
        expect.objectContaining({ passive: true })
      )
      expect(addSpy).toHaveBeenCalledWith(
        "keydown",
        expect.any(Function),
        expect.objectContaining({ passive: true })
      )
      expect(addSpy).toHaveBeenCalledWith(
        "touchstart",
        expect.any(Function),
        expect.objectContaining({ passive: true })
      )
      expect(addSpy).toHaveBeenCalledWith(
        "scroll",
        expect.any(Function),
        expect.objectContaining({ passive: true })
      )
      expect(addSpy).toHaveBeenCalledWith(
        "click",
        expect.any(Function),
        expect.objectContaining({ passive: true })
      )
    })

    addSpy.mockRestore()
  })

  it("shows warning toast at 25 minutes of inactivity", async () => {
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue("token")
    vi.mocked(apiClient.apiGetMe).mockResolvedValue({
      data: mockUser,
      error: null,
    })
    vi.mocked(apiClient.apiGetWorkspaces).mockResolvedValue({
      data: [mockWorkspace],
      error: null,
    })
    vi.mocked(apiClient.getStoredWorkspaceId).mockReturnValue(null)

    renderHook(() => useAuth(), { wrapper: createWrapper() })

    await waitFor(() => {
      // Wait for auth to load
    })

    // Advance to 25 minutes (warning time)
    await act(async () => {
      vi.advanceTimersByTime(25 * 60 * 1000)
    })

    expect(mockToastWarning).toHaveBeenCalledWith(
      "You will be logged out in 5 minutes due to inactivity.",
      expect.objectContaining({ duration: 5 * 60 * 1000 })
    )
  })

  it("calls logout after 30 minutes of inactivity", async () => {
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue("token")
    vi.mocked(apiClient.getStoredRefreshToken).mockReturnValue("refresh-token")
    vi.mocked(apiClient.apiGetMe).mockResolvedValue({
      data: mockUser,
      error: null,
    })
    vi.mocked(apiClient.apiGetWorkspaces).mockResolvedValue({
      data: [mockWorkspace],
      error: null,
    })
    vi.mocked(apiClient.apiLogout).mockResolvedValue({
      data: null,
      error: null,
    })
    vi.mocked(apiClient.getStoredWorkspaceId).mockReturnValue(null)

    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true)
    })

    // Advance to 30 minutes (logout time)
    await act(async () => {
      vi.advanceTimersByTime(30 * 60 * 1000)
    })

    // User should be logged out
    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.user).toBeNull()
    expect(apiClient.clearTokens).toHaveBeenCalled()
  })

  it("resets timer on user activity", async () => {
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue("token")
    vi.mocked(apiClient.getStoredRefreshToken).mockReturnValue("refresh-token")
    vi.mocked(apiClient.apiGetMe).mockResolvedValue({
      data: mockUser,
      error: null,
    })
    vi.mocked(apiClient.apiGetWorkspaces).mockResolvedValue({
      data: [mockWorkspace],
      error: null,
    })
    vi.mocked(apiClient.getStoredWorkspaceId).mockReturnValue(null)

    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true)
    })

    // Advance 20 minutes
    await act(async () => {
      vi.advanceTimersByTime(20 * 60 * 1000)
    })

    // Simulate user activity - reset the timer
    await act(async () => {
      window.dispatchEvent(new Event("mousemove"))
    })

    // Advance another 20 minutes (total 40 from start, but only 20 from activity)
    await act(async () => {
      vi.advanceTimersByTime(20 * 60 * 1000)
    })

    // Should still be authenticated because the timer was reset
    expect(result.current.isAuthenticated).toBe(true)
  })

  it("cleans up event listeners on unmount", async () => {
    const removeSpy = vi.spyOn(window, "removeEventListener")

    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue("token")
    vi.mocked(apiClient.apiGetMe).mockResolvedValue({
      data: mockUser,
      error: null,
    })
    vi.mocked(apiClient.apiGetWorkspaces).mockResolvedValue({
      data: [mockWorkspace],
      error: null,
    })
    vi.mocked(apiClient.getStoredWorkspaceId).mockReturnValue(null)

    const { unmount } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {})

    unmount()

    expect(removeSpy).toHaveBeenCalledWith(
      "mousemove",
      expect.any(Function)
    )
    expect(removeSpy).toHaveBeenCalledWith(
      "keydown",
      expect.any(Function)
    )
    expect(removeSpy).toHaveBeenCalledWith(
      "touchstart",
      expect.any(Function)
    )
    expect(removeSpy).toHaveBeenCalledWith(
      "scroll",
      expect.any(Function)
    )
    expect(removeSpy).toHaveBeenCalledWith(
      "click",
      expect.any(Function)
    )

    removeSpy.mockRestore()
  })

  it("does not set up inactivity timer when not authenticated", async () => {
    const addSpy = vi.spyOn(window, "addEventListener")

    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue(null)

    renderHook(() => useAuth(), { wrapper: createWrapper() })

    await waitFor(() => {})

    // Should not register activity events (only the global ones from setup may be there)
    const activityCalls = addSpy.mock.calls.filter(
      ([event]) =>
        event === "mousemove" ||
        event === "keydown" ||
        event === "touchstart" ||
        event === "scroll" ||
        event === "click"
    )
    expect(activityCalls).toHaveLength(0)

    addSpy.mockRestore()
  })
})
