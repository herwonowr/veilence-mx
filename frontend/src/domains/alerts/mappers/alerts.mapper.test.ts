import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import {
  toAlertNoteViewModel,
  toAlertNoteViewModels,
} from "@/domains/alerts/mappers/alerts.mapper"
import type { AlertNote } from "@/domains/alerts/types/alerts.types"

const makeNote = (overrides: Partial<AlertNote> = {}): AlertNote => ({
  id: 1,
  alertId: 10,
  userId: 100,
  userEmail: "john.doe@example.com",
  content: "This looks suspicious.",
  createdAt: "2026-04-15T10:00:00Z",
  updatedAt: "2026-04-15T10:00:00Z",
  ...overrides,
})

describe("toAlertNoteViewModel", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-04-15T12:00:00Z"))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("maps all fields correctly", () => {
    const note = makeNote()
    const vm = toAlertNoteViewModel(note)

    expect(vm.id).toBe(1)
    expect(vm.alertId).toBe(10)
    expect(vm.userId).toBe(100)
    expect(vm.userEmail).toBe("john.doe@example.com")
    expect(vm.content).toBe("This looks suspicious.")
    expect(vm.createdAt).toBe("2026-04-15T10:00:00Z")
    expect(vm.updatedAt).toBe("2026-04-15T10:00:00Z")
  })

  it("detects isEdited when updatedAt is >1s after createdAt", () => {
    const note = makeNote({
      createdAt: "2026-04-15T10:00:00Z",
      updatedAt: "2026-04-15T10:05:00Z",
    })
    const vm = toAlertNoteViewModel(note)
    expect(vm.isEdited).toBe(true)
  })

  it("sets isEdited to false when updatedAt equals createdAt", () => {
    const note = makeNote({
      createdAt: "2026-04-15T10:00:00Z",
      updatedAt: "2026-04-15T10:00:00Z",
    })
    const vm = toAlertNoteViewModel(note)
    expect(vm.isEdited).toBe(false)
  })

  it("sets isEdited to false when updatedAt is within 1s of createdAt", () => {
    const note = makeNote({
      createdAt: "2026-04-15T10:00:00.000Z",
      updatedAt: "2026-04-15T10:00:00.500Z",
    })
    const vm = toAlertNoteViewModel(note)
    expect(vm.isEdited).toBe(false)
  })

  describe("userInitials (getInitials)", () => {
    it("extracts initials from dotted email prefix", () => {
      const vm = toAlertNoteViewModel(
        makeNote({ userEmail: "john.doe@example.com" })
      )
      expect(vm.userInitials).toBe("JD")
    })

    it("extracts initials from hyphenated email prefix", () => {
      const vm = toAlertNoteViewModel(
        makeNote({ userEmail: "jane-smith@example.com" })
      )
      expect(vm.userInitials).toBe("JS")
    })

    it("extracts initials from underscored email prefix", () => {
      const vm = toAlertNoteViewModel(
        makeNote({ userEmail: "bob_jones@example.com" })
      )
      expect(vm.userInitials).toBe("BJ")
    })

    it("uses first two chars for single-part email prefix", () => {
      const vm = toAlertNoteViewModel(
        makeNote({ userEmail: "admin@example.com" })
      )
      expect(vm.userInitials).toBe("AD")
    })

    it("returns '?' for empty email", () => {
      const vm = toAlertNoteViewModel(makeNote({ userEmail: "" }))
      expect(vm.userInitials).toBe("?")
    })

    it("handles single-char email prefix", () => {
      const vm = toAlertNoteViewModel(
        makeNote({ userEmail: "a@example.com" })
      )
      expect(vm.userInitials).toBe("A")
    })
  })

  describe("relativeTime (formatRelativeTime)", () => {
    it("returns 'just now' for very recent timestamps", () => {
      const note = makeNote({ createdAt: "2026-04-15T12:00:00Z" })
      const vm = toAlertNoteViewModel(note)
      expect(vm.relativeTime).toBe("just now")
    })

    it("returns minutes ago for timestamps within an hour", () => {
      const note = makeNote({ createdAt: "2026-04-15T11:45:00Z" })
      const vm = toAlertNoteViewModel(note)
      expect(vm.relativeTime).toBe("15m ago")
    })

    it("returns hours ago for timestamps within a day", () => {
      const note = makeNote({ createdAt: "2026-04-15T10:00:00Z" })
      const vm = toAlertNoteViewModel(note)
      expect(vm.relativeTime).toBe("2h ago")
    })

    it("returns days ago for timestamps within a week", () => {
      const note = makeNote({ createdAt: "2026-04-13T12:00:00Z" })
      const vm = toAlertNoteViewModel(note)
      expect(vm.relativeTime).toBe("2d ago")
    })

    it("returns formatted date for timestamps older than a week", () => {
      const note = makeNote({ createdAt: "2026-04-01T12:00:00Z" })
      const vm = toAlertNoteViewModel(note)
      // Falls back to toLocaleDateString - just verify it doesn't return relative format
      expect(vm.relativeTime).not.toContain("ago")
      expect(vm.relativeTime).not.toBe("just now")
    })

    it("returns 'just now' for future timestamps", () => {
      const note = makeNote({ createdAt: "2026-04-15T13:00:00Z" })
      const vm = toAlertNoteViewModel(note)
      expect(vm.relativeTime).toBe("just now")
    })
  })
})

describe("toAlertNoteViewModels", () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-04-15T12:00:00Z"))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("maps an array of notes to view models", () => {
    const notes = [
      makeNote({ id: 1, userEmail: "alice.b@example.com" }),
      makeNote({ id: 2, userEmail: "charlie@example.com" }),
    ]
    const vms = toAlertNoteViewModels(notes)

    expect(vms).toHaveLength(2)
    expect(vms[0]!.id).toBe(1)
    expect(vms[0]!.userInitials).toBe("AB")
    expect(vms[1]!.id).toBe(2)
    expect(vms[1]!.userInitials).toBe("CH")
  })

  it("returns empty array for empty input", () => {
    expect(toAlertNoteViewModels([])).toEqual([])
  })
})
