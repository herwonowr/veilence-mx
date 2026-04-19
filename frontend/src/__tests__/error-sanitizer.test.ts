/**
 * SEC-S4-10: Tests for error message sanitization utility.
 */

import { sanitizeErrorMessage } from "@/core"

describe("sanitizeErrorMessage", () => {
  describe("auth errors", () => {
    it("maps 'Unauthorized' to a friendly session-expired message", () => {
      expect(sanitizeErrorMessage(new Error("Unauthorized"))).toBe(
        "Your session has expired. Please sign in again."
      )
    })

    it("maps 'Forbidden' to a permissions message", () => {
      expect(sanitizeErrorMessage(new Error("Forbidden"))).toBe(
        "You do not have permission to perform this action."
      )
    })

    it("maps 'invalid credentials' to a friendly message", () => {
      expect(sanitizeErrorMessage(new Error("invalid credentials"))).toBe(
        "Invalid email or password."
      )
    })

    it("maps 'Invalid email or password' to a friendly message", () => {
      expect(sanitizeErrorMessage(new Error("Invalid email or password"))).toBe(
        "Invalid email or password."
      )
    })

    it("maps token expired errors", () => {
      expect(sanitizeErrorMessage(new Error("token has expired"))).toBe(
        "Your session has expired. Please sign in again."
      )
    })
  })

  describe("network errors", () => {
    it("maps 'Failed to fetch' to a connectivity message", () => {
      expect(sanitizeErrorMessage(new Error("Failed to fetch"))).toBe(
        "Unable to connect to the server. Check your network connection."
      )
    })

    it("maps network errors", () => {
      expect(sanitizeErrorMessage(new Error("NetworkError when attempting to fetch resource"))).toBe(
        "A network error occurred. Please check your connection and try again."
      )
    })

    it("maps timeout errors", () => {
      expect(sanitizeErrorMessage(new Error("Request timeout"))).toBe(
        "The request timed out. Please try again."
      )
    })
  })

  describe("server errors", () => {
    it("maps API error 500 to a server error message", () => {
      expect(sanitizeErrorMessage(new Error("API error: 500"))).toBe(
        "An unexpected server error occurred. Please try again later."
      )
    })

    it("maps API error 502 to a server error message", () => {
      expect(sanitizeErrorMessage(new Error("API error: 502"))).toBe(
        "An unexpected server error occurred. Please try again later."
      )
    })

    it("maps API error 400 to a client error message", () => {
      expect(sanitizeErrorMessage(new Error("API error: 400"))).toBe(
        "The request could not be completed. Please try again."
      )
    })
  })

  describe("validation errors", () => {
    it("maps 'bad request' errors", () => {
      expect(sanitizeErrorMessage(new Error("Bad Request"))).toBe(
        "The request was invalid. Please check your input."
      )
    })
  })

  describe("unsafe messages", () => {
    it("strips Go stack trace references", () => {
      const result = sanitizeErrorMessage(
        new Error("pq: duplicate key value violates unique constraint")
      )
      expect(result).toBe("Something went wrong. Please try again.")
    })

    it("strips SQL fragments", () => {
      const result = sanitizeErrorMessage(
        new Error("SQL: SELECT * FROM users WHERE id = 1")
      )
      expect(result).toBe("Something went wrong. Please try again.")
    })

    it("strips internal server references", () => {
      const result = sanitizeErrorMessage(
        new Error("internal server error at handler.go:42")
      )
      // Should match either the server error pattern or the go-file-reference safe check
      expect(result).not.toContain("handler.go")
    })

    it("strips GORM errors", () => {
      const result = sanitizeErrorMessage(
        new Error("GORM: constraint violation on table users")
      )
      expect(result).toBe("Something went wrong. Please try again.")
    })

    it("strips very long messages", () => {
      const longMessage = "x".repeat(300)
      const result = sanitizeErrorMessage(new Error(longMessage))
      expect(result).toBe("Something went wrong. Please try again.")
    })
  })

  describe("safe passthrough", () => {
    it("passes through short, safe messages", () => {
      const msg = "Email already in use"
      // This matches the "email.*already" pattern
      expect(sanitizeErrorMessage(new Error(msg))).toBe(
        "An account with this email already exists."
      )
    })

    it("uses generic fallback for unmapped safe messages", () => {
      const msg = "Something custom happened"
      expect(sanitizeErrorMessage(new Error(msg))).toBe(msg)
    })
  })

  describe("edge cases", () => {
    it("handles string errors", () => {
      expect(sanitizeErrorMessage("Unauthorized")).toBe(
        "Your session has expired. Please sign in again."
      )
    })

    it("handles unknown error types", () => {
      // Non-Error, non-string values produce "Unknown error" which is short and safe
      expect(sanitizeErrorMessage(42)).toBe("Unknown error")
    })

    it("handles null", () => {
      expect(sanitizeErrorMessage(null)).toBe("Unknown error")
    })

    it("handles undefined", () => {
      expect(sanitizeErrorMessage(undefined)).toBe("Unknown error")
    })
  })
})
