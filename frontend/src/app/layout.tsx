import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { ThemeProvider } from "@/core/providers/theme-provider";
import { AuthProvider } from "@/core/providers/auth-provider";
import { QueryProvider } from "@/core/providers/query-provider";
import { AppShell } from "@/features/shell";
import { NotificationBell } from "@/features/notifications";
import { OrgSelector } from "@/features/admin";
import { Toaster } from "@/ui/feedback/toaster";
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

const RootLayout = ({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) => {
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
                orgSelectorSlot={<OrgSelector />}
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