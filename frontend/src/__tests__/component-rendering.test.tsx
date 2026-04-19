/**
 * Component integration test for the Skeleton loading component
 * and a representative page component rendered with mock data.
 */

import { render, screen, within } from "@/test-utils"
import { Skeleton } from "@/ui"

// Mock auth-context to provide authenticated state
vi.mock("@/core/providers/auth-provider", () => ({
  useAuth: vi.fn(() => ({
    user: {
      id: 1,
      email: "test@example.com",
      firstName: "Test",
      lastName: "User",
      isActive: true,
      emailVerified: true,
      lastLoginAt: null,
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
    },
    isAuthenticated: true,
    isLoading: false,
    currentWorkspace: {
      id: 1,
      name: "Test Org",
      slug: "test-org",
      description: "Testing org",
      ownerId: 1,
      isActive: true,
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
    },
    workspaces: [],
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    setCurrentWorkspace: vi.fn(),
    refreshUser: vi.fn(),
    refreshWorkspaces: vi.fn(),
  })),
  AuthProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

// ── A representative dashboard summary card component ──
// This simulates the kind of component the frontend team will build.
// Tests verify that data flows correctly through props to rendered output.

interface StatCardProps {
  title: string
  value: number | string
  description?: string
  trend?: "up" | "down" | "neutral"
}

const StatCard = ({ title, value, description, trend }: StatCardProps) => {
  return (
    <div data-testid={`stat-card-${title.toLowerCase().replace(/\s+/g, "-")}`}>
      <h3>{title}</h3>
      <p data-testid="stat-value">{value}</p>
      {description && <p data-testid="stat-description">{description}</p>}
      {trend && <span data-testid="stat-trend" data-trend={trend} />}
    </div>
  )
}

interface AlertRowProps {
  id: number
  severity: "low" | "medium" | "high" | "critical"
  message: string
  packageName: string
  status: string
}

const AlertRow = ({ id, severity, message, packageName, status }: AlertRowProps) => {
  return (
    <tr data-testid={`alert-row-${id}`}>
      <td data-testid="alert-severity">{severity}</td>
      <td data-testid="alert-package">{packageName}</td>
      <td data-testid="alert-message">{message}</td>
      <td data-testid="alert-status">{status}</td>
    </tr>
  )
}

const AlertsTable = ({ alerts }: { alerts: AlertRowProps[] }) => {
  if (alerts.length === 0) {
    return <p data-testid="empty-state">No alerts found</p>
  }

  return (
    <table data-testid="alerts-table">
      <thead>
        <tr>
          <th>Severity</th>
          <th>Package</th>
          <th>Message</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        {alerts.map((alert) => (
          <AlertRow key={alert.id} {...alert} />
        ))}
      </tbody>
    </table>
  )
}

describe("Skeleton component", () => {
  it("renders with correct className", () => {
    const { container } = render(<Skeleton className="h-8 w-48" />, {
      wrapper: ({ children }) => <>{children}</>,
    })

    const skeleton = container.firstElementChild
    expect(skeleton).toBeInTheDocument()
    expect(skeleton?.className).toContain("h-8")
    expect(skeleton?.className).toContain("w-48")
  })
})

describe("StatCard component", () => {
  it("renders title and value", () => {
    render(<StatCard title="Total Packages" value={150} />, {
      wrapper: ({ children }) => <>{children}</>,
    })

    expect(screen.getByText("Total Packages")).toBeInTheDocument()
    expect(screen.getByTestId("stat-value")).toHaveTextContent("150")
  })

  it("renders optional description", () => {
    render(
      <StatCard title="Active Alerts" value={3} description="+2 from last week" />,
      { wrapper: ({ children }) => <>{children}</> }
    )

    expect(screen.getByTestId("stat-description")).toHaveTextContent(
      "+2 from last week"
    )
  })

  it("renders trend indicator", () => {
    render(<StatCard title="Releases" value={1200} trend="up" />, {
      wrapper: ({ children }) => <>{children}</>,
    })

    const trend = screen.getByTestId("stat-trend")
    expect(trend).toHaveAttribute("data-trend", "up")
  })

  it("does not render description when not provided", () => {
    render(<StatCard title="Packages" value={100} />, {
      wrapper: ({ children }) => <>{children}</>,
    })

    expect(screen.queryByTestId("stat-description")).not.toBeInTheDocument()
  })
})

describe("AlertsTable component", () => {
  const mockAlerts: AlertRowProps[] = [
    {
      id: 1,
      severity: "critical",
      message: "Malicious code detected in install script",
      packageName: "evil-pkg",
      status: "new",
    },
    {
      id: 2,
      severity: "high",
      message: "Suspicious obfuscated code",
      packageName: "shady-lib",
      status: "acknowledged",
    },
    {
      id: 3,
      severity: "low",
      message: "Minor metadata change",
      packageName: "safe-pkg",
      status: "resolved",
    },
  ]

  it("renders all alert rows", () => {
    render(<AlertsTable alerts={mockAlerts} />, {
      wrapper: ({ children }) => <>{children}</>,
    })

    expect(screen.getByTestId("alerts-table")).toBeInTheDocument()
    expect(screen.getByTestId("alert-row-1")).toBeInTheDocument()
    expect(screen.getByTestId("alert-row-2")).toBeInTheDocument()
    expect(screen.getByTestId("alert-row-3")).toBeInTheDocument()
  })

  it("displays correct severity for each alert", () => {
    render(<AlertsTable alerts={mockAlerts} />, {
      wrapper: ({ children }) => <>{children}</>,
    })

    const row1 = screen.getByTestId("alert-row-1")
    expect(within(row1).getByTestId("alert-severity")).toHaveTextContent(
      "critical"
    )

    const row3 = screen.getByTestId("alert-row-3")
    expect(within(row3).getByTestId("alert-severity")).toHaveTextContent("low")
  })

  it("shows empty state when no alerts", () => {
    render(<AlertsTable alerts={[]} />, {
      wrapper: ({ children }) => <>{children}</>,
    })

    expect(screen.getByTestId("empty-state")).toBeInTheDocument()
    expect(screen.getByText("No alerts found")).toBeInTheDocument()
    expect(screen.queryByTestId("alerts-table")).not.toBeInTheDocument()
  })

  it("displays package name and message correctly", () => {
    render(<AlertsTable alerts={[mockAlerts[0]]} />, {
      wrapper: ({ children }) => <>{children}</>,
    })

    const row = screen.getByTestId("alert-row-1")
    expect(within(row).getByTestId("alert-package")).toHaveTextContent(
      "evil-pkg"
    )
    expect(within(row).getByTestId("alert-message")).toHaveTextContent(
      "Malicious code detected in install script"
    )
  })
})
