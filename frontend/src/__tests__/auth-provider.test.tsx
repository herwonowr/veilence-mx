import { renderHook, act, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { AuthProvider, useAuth } from "@/core"
import * as apiClient from "@/core"

// Wrapper that provides both QueryClient and AuthProvider
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  // eslint-disable-next-line react/display-name
  return ({ children }: { children: React.ReactNode }) => {
    return (
      <QueryClientProvider client={queryClient}>
        <AuthProvider>{children}</AuthProvider>
      </QueryClientProvider>
    )
  }
}

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

describe("AuthProvider", () => {
  it("throws when useAuth is used outside AuthProvider", () => {
    // Suppress React error boundary console output
    const spy = vi.spyOn(console, "error").mockImplementation(() => {})

    expect(() => {
      renderHook(() => useAuth())
    }).toThrow("useAuth must be used within an AuthProvider")

    spy.mockRestore()
  })

  it("starts with unauthenticated state when no stored token", async () => {
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue(null)

    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.user).toBeNull()
  })

  it("restores auth state from stored token on mount", async () => {
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue("stored-token")
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
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.isAuthenticated).toBe(true)
    expect(result.current.user).toEqual(mockUser)
    expect(result.current.workspaces).toEqual([mockWorkspace])
    expect(result.current.currentWorkspace).toEqual(mockWorkspace)
  })

  it("handles login flow correctly", async () => {
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue(null)
    vi.mocked(apiClient.apiLogin).mockResolvedValue({
      data: {
        user: mockUser,
        accessToken: "new-access-token",
        refreshToken: "new-refresh-token",
      },
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
      expect(result.current.isLoading).toBe(false)
    })

    // Not authenticated initially
    expect(result.current.isAuthenticated).toBe(false)

    // Perform login
    await act(async () => {
      await result.current.login("test@example.com", "password123")
    })

    // Should now be authenticated
    expect(result.current.isAuthenticated).toBe(true)
    expect(result.current.user).toEqual(mockUser)
    expect(apiClient.storeTokens).toHaveBeenCalledWith(
      "new-access-token",
      "new-refresh-token"
    )
  })

  it("handles logout flow correctly", async () => {
    // Start authenticated
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue("stored-token")
    vi.mocked(apiClient.getStoredRefreshToken).mockReturnValue("stored-refresh")
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

    // Perform logout
    await act(async () => {
      await result.current.logout()
    })

    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.user).toBeNull()
    expect(result.current.currentWorkspace).toBeNull()
    expect(result.current.workspaces).toEqual([])
    expect(apiClient.clearTokens).toHaveBeenCalled()
    expect(apiClient.clearWorkspaceId).toHaveBeenCalled()
  })

  it("sets current workspace and persists to localStorage", async () => {
    vi.mocked(apiClient.getStoredAccessToken).mockReturnValue("stored-token")
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
      expect(result.current.isLoading).toBe(false)
    })

    const newWorkspace = { ...mockWorkspace, id: 2, name: "New Workspace", slug: "new-workspace" }

    act(() => {
      result.current.setCurrentWorkspace(newWorkspace)
    })

    expect(result.current.currentWorkspace).toEqual(newWorkspace)
    expect(apiClient.storeWorkspaceId).toHaveBeenCalledWith(2)
  })
})
