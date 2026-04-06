/**
 * Tests for the api-client fetchApi wrapper, specifically the
 * 401 → token-refresh → retry flow and auth header attachment.
 */

// We test the exported functions which use the internal fetchApi wrapper.
// Each test group uses dynamic imports with vi.resetModules() to get
// a fresh module instance with a clean isRefreshing state.

beforeEach(() => {
  localStorage.clear()
  // Clear _csrf_token cookie reliably across test environments
  document.cookie = "_csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/"
  document.cookie = "_csrf_token=; expires=Thu, 01 Jan 1970 00:00:00 GMT"
  // Force-overwrite with empty value in case happy-dom doesn't support expires
  Object.defineProperty(document, "cookie", {
    writable: true,
    value: "",
  })
  vi.stubGlobal("fetch", vi.fn())
})

async function loadApiClient() {
  vi.resetModules()
  return import("@/lib/api-client")
}

describe("api-client", () => {
  describe("token storage", () => {
    it("stores and retrieves tokens from localStorage", async () => {
      const api = await loadApiClient()

      api.storeTokens("access-123", "refresh-456")

      expect(localStorage.getItem("vmx_access_token")).toBe("access-123")
      expect(localStorage.getItem("vmx_refresh_token")).toBe("refresh-456")

      expect(api.getStoredAccessToken()).toBe("access-123")
      expect(api.getStoredRefreshToken()).toBe("refresh-456")
    })

    it("clears tokens from localStorage", async () => {
      const api = await loadApiClient()

      api.storeTokens("access-123", "refresh-456")
      api.clearTokens()

      expect(api.getStoredAccessToken()).toBeNull()
      expect(api.getStoredRefreshToken()).toBeNull()
    })

    it("stores and retrieves org ID", async () => {
      const api = await loadApiClient()

      api.storeOrgId(42)
      expect(api.getStoredOrgId()).toBe(42)

      api.clearOrgId()
      expect(api.getStoredOrgId()).toBeNull()
    })
  })

  describe("fetchApi auth headers", () => {
    it("attaches Authorization and X-Org-ID headers", async () => {
      const api = await loadApiClient()

      // Store tokens before making the request
      localStorage.setItem("vmx_access_token", "my-token")
      localStorage.setItem("vmx_current_org_id", "5")

      const mockResponse = {
        ok: true,
        status: 200,
        json: async () => ({ data: { totalPackages: 10 }, error: null }),
      }
      vi.mocked(fetch).mockResolvedValueOnce(mockResponse as Response)

      await api.getDashboardStats()

      expect(fetch).toHaveBeenCalledTimes(1)
      const [url, options] = vi.mocked(fetch).mock.calls[0]
      expect(url).toBe("http://localhost:8080/api/dashboard/stats")
      expect((options?.headers as Record<string, string>)["Authorization"]).toBe(
        "Bearer my-token"
      )
      expect((options?.headers as Record<string, string>)["X-Org-ID"]).toBe("5")
    })

    it("skips auth headers for login requests", async () => {
      const api = await loadApiClient()

      localStorage.setItem("vmx_access_token", "my-token")

      const mockResponse = {
        ok: true,
        status: 200,
        json: async () => ({
          data: {
            user: { id: 1, email: "test@test.com" },
            accessToken: "new-token",
            refreshToken: "new-refresh",
          },
          error: null,
        }),
      }
      vi.mocked(fetch).mockResolvedValueOnce(mockResponse as Response)

      await api.apiLogin("test@test.com", "password")

      const [, options] = vi.mocked(fetch).mock.calls[0]
      expect(
        (options?.headers as Record<string, string>)["Authorization"]
      ).toBeUndefined()
    })
  })

  describe("401 token refresh flow", () => {
    it("retries request with new token after successful refresh", async () => {
      const api = await loadApiClient()

      localStorage.setItem("vmx_access_token", "expired-token")
      localStorage.setItem("vmx_refresh_token", "valid-refresh-token")

      // First call: 401
      const unauthorizedResponse = {
        ok: false,
        status: 401,
        json: async () => ({ data: null, error: "Unauthorized" }),
      }

      // Refresh call: success
      const refreshResponse = {
        ok: true,
        status: 200,
        json: async () => ({
          data: {
            accessToken: "new-access-token",
            refreshToken: "new-refresh-token",
            user: { id: 1 },
          },
          error: null,
        }),
      }

      // Retry call: success
      const successResponse = {
        ok: true,
        status: 200,
        json: async () => ({
          data: { totalPackages: 42 },
          error: null,
        }),
      }

      vi.mocked(fetch)
        .mockResolvedValueOnce(unauthorizedResponse as Response) // initial call → 401
        .mockResolvedValueOnce(refreshResponse as Response) // refresh token call
        .mockResolvedValueOnce(successResponse as Response) // retry call

      const result = await api.getDashboardStats()

      expect(result.data.totalPackages).toBe(42)

      // 3 fetch calls: original, refresh, retry
      expect(fetch).toHaveBeenCalledTimes(3)

      // Verify refresh was called with the right body
      const [refreshUrl, refreshOptions] = vi.mocked(fetch).mock.calls[1]
      expect(refreshUrl).toBe("http://localhost:8080/api/auth/refresh")
      expect(JSON.parse(refreshOptions?.body as string)).toEqual({
        refreshToken: "valid-refresh-token",
      })

      // Verify the retry used the new token
      const [, retryOptions] = vi.mocked(fetch).mock.calls[2]
      expect(
        (retryOptions?.headers as Record<string, string>)["Authorization"]
      ).toBe("Bearer new-access-token")
    })

    it("throws error when refresh fails", async () => {
      const api = await loadApiClient()

      localStorage.setItem("vmx_access_token", "expired-token")
      localStorage.setItem("vmx_refresh_token", "invalid-refresh-token")

      // First call: 401
      const unauthorizedResponse = {
        ok: false,
        status: 401,
        json: async () => ({ data: null, error: "Unauthorized" }),
      }

      // Refresh call: also fails
      const refreshFailResponse = {
        ok: false,
        status: 401,
        json: async () => ({
          data: null,
          error: "Invalid refresh token",
        }),
      }

      vi.mocked(fetch)
        .mockResolvedValueOnce(unauthorizedResponse as Response)
        .mockResolvedValueOnce(refreshFailResponse as Response)

      await expect(api.getDashboardStats()).rejects.toThrow(
        "Your session has expired. Please sign in again."
      )
    })

    it("does not attempt refresh when no refresh token is stored", async () => {
      const api = await loadApiClient()

      localStorage.setItem("vmx_access_token", "expired-token")
      // No refresh token stored

      const unauthorizedResponse = {
        ok: false,
        status: 401,
        json: async () => ({ data: null, error: "Unauthorized" }),
      }

      vi.mocked(fetch).mockResolvedValueOnce(unauthorizedResponse as Response)

      await expect(api.getDashboardStats()).rejects.toThrow(
        "Your session has expired. Please sign in again."
      )

      // Only the original call, no refresh attempt
      expect(fetch).toHaveBeenCalledTimes(1)
    })
  })

  describe("error handling", () => {
    it("throws with sanitized API error message on non-ok response", async () => {
      const api = await loadApiClient()

      const errorResponse = {
        ok: false,
        status: 403,
        json: async () => ({ data: null, error: "Forbidden" }),
      }
      vi.mocked(fetch).mockResolvedValueOnce(errorResponse as Response)

      // SEC-S4-10: Error messages are now sanitized
      await expect(api.getDashboardStats()).rejects.toThrow(
        "You do not have permission to perform this action."
      )
    })

    it("throws with sanitized message when no error message provided", async () => {
      const api = await loadApiClient()

      const errorResponse = {
        ok: false,
        status: 500,
        json: async () => ({ data: null, error: null }),
      }
      vi.mocked(fetch).mockResolvedValueOnce(errorResponse as Response)

      // SEC-S4-10: "API error: 500" is sanitized to a user-friendly message
      await expect(api.getDashboardStats()).rejects.toThrow(
        "An unexpected server error occurred. Please try again later."
      )
    })
  })

  describe("CSRF token header (SEC-S4-002)", () => {
    it("attaches X-CSRF-Token header for POST requests when cookie is set", async () => {
      const api = await loadApiClient()

      document.cookie = "_csrf_token=abc123csrftoken"

      const mockResponse = {
        ok: true,
        status: 200,
        json: async () => ({
          data: { name: "test-package", registry: "npm" },
          error: null,
        }),
      }
      vi.mocked(fetch).mockResolvedValueOnce(mockResponse as Response)

      await api.createPackage("test-package", "npm")

      const [, options] = vi.mocked(fetch).mock.calls[0]
      expect(
        (options?.headers as Record<string, string>)["X-CSRF-Token"]
      ).toBe("abc123csrftoken")
    })

    it("attaches X-CSRF-Token header for DELETE requests", async () => {
      const api = await loadApiClient()

      document.cookie = "_csrf_token=delete-csrf-tok"

      const mockResponse = {
        ok: true,
        status: 200,
        json: async () => ({ data: null, error: null }),
      }
      vi.mocked(fetch).mockResolvedValueOnce(mockResponse as Response)

      await api.deletePackage(1)

      const [, options] = vi.mocked(fetch).mock.calls[0]
      expect(
        (options?.headers as Record<string, string>)["X-CSRF-Token"]
      ).toBe("delete-csrf-tok")
    })

    it("attaches X-CSRF-Token header for PUT requests", async () => {
      const api = await loadApiClient()

      document.cookie = "_csrf_token=put-csrf-tok"

      const mockResponse = {
        ok: true,
        status: 200,
        json: async () => ({
          data: { id: 1, status: "acknowledged" },
          error: null,
        }),
      }
      vi.mocked(fetch).mockResolvedValueOnce(mockResponse as Response)

      await api.updateSettings({ key: "value" })

      const [, options] = vi.mocked(fetch).mock.calls[0]
      expect(
        (options?.headers as Record<string, string>)["X-CSRF-Token"]
      ).toBe("put-csrf-tok")
    })

    it("attaches X-CSRF-Token header for PATCH requests", async () => {
      const api = await loadApiClient()

      document.cookie = "_csrf_token=patch-csrf-tok"

      const mockResponse = {
        ok: true,
        status: 200,
        json: async () => ({
          data: { id: 1, status: "acknowledged" },
          error: null,
        }),
      }
      vi.mocked(fetch).mockResolvedValueOnce(mockResponse as Response)

      await api.updateAlertStatus(1, "acknowledged")

      const [, options] = vi.mocked(fetch).mock.calls[0]
      expect(
        (options?.headers as Record<string, string>)["X-CSRF-Token"]
      ).toBe("patch-csrf-tok")
    })

    it("does NOT attach X-CSRF-Token header for GET requests", async () => {
      const api = await loadApiClient()

      document.cookie = "_csrf_token=should-not-appear"

      const mockResponse = {
        ok: true,
        status: 200,
        json: async () => ({ data: { totalPackages: 10 }, error: null }),
      }
      vi.mocked(fetch).mockResolvedValueOnce(mockResponse as Response)

      await api.getDashboardStats()

      const [, options] = vi.mocked(fetch).mock.calls[0]
      expect(
        (options?.headers as Record<string, string>)["X-CSRF-Token"]
      ).toBeUndefined()
    })

    it("does NOT attach X-CSRF-Token header when cookie is absent", async () => {
      const api = await loadApiClient()

      // No csrf_token cookie set

      const mockResponse = {
        ok: true,
        status: 200,
        json: async () => ({
          data: { name: "test-package", registry: "npm" },
          error: null,
        }),
      }
      vi.mocked(fetch).mockResolvedValueOnce(mockResponse as Response)

      await api.createPackage("test-package", "npm")

      const [, options] = vi.mocked(fetch).mock.calls[0]
      expect(
        (options?.headers as Record<string, string>)["X-CSRF-Token"]
      ).toBeUndefined()
    })
  })
})
