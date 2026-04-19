import type { Metadata } from "next";
import { cookies } from "next/headers";
import { Geist, Geist_Mono } from "next/font/google";
import { ThemeProvider, AuthProvider, QueryProvider } from "@/core";
import { AppShell } from "@/features/shell";
import { NotificationBell } from "@/features/notifications";
import { WorkspaceSelector } from "@/features/admin";
import { Toaster } from "@/ui";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Veilence-MX",
  description: "Supply Chain Compromise Monitor for Python and NPM",
};

const RootLayout = async ({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) => {
  const cookieStore = await cookies();
  const sidebarCookie = cookieStore.get("sidebar_state");
  const sidebarDefaultOpen = sidebarCookie?.value !== "false";

  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
      suppressHydrationWarning
    >
      <body className="min-h-full">
        <a
          href="#main-content"
          className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-primary focus:px-4 focus:py-2 focus:text-primary-foreground focus:outline-none"
        >
          Skip to main content
        </a>
        <ThemeProvider>
          <QueryProvider>
            <AuthProvider>
              <AppShell
                notificationSlot={<NotificationBell />}
                workspaceSelectorSlot={<WorkspaceSelector />}
                sidebarDefaultOpen={sidebarDefaultOpen}
              >
                {children}
              </AppShell>
              <Toaster />
            </AuthProvider>
          </QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
export default RootLayout