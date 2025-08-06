import type { Metadata } from 'next'
import { Suspense } from 'react'
import { AuthGuard } from '@/components/auth'
import './globals.css'

export const metadata: Metadata = {
  title: 'CCDash - Claude Code Dashboard',
  description: 'Monitor Claude Code usage, manage sessions, and track development activities. Comprehensive dashboard for token consumption, session management, and AI development workflow optimization.',
  keywords: ['Claude Code', 'AI Development', 'Dashboard', 'Monitoring', 'Development Tools'],
  generator: 'React + Next.js + TailwindCSS + TypeScript + Go',
  authors: [{ name: 'CCDash Team' }],
  icons: {
    icon: '/favicon.ico',
    shortcut: '/favicon.ico',
    apple: '/favicon.ico',
  },
}

export const viewport = {
  width: 'device-width',
  initialScale: 1,
  themeColor: '#000000',
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-background font-sans antialiased">
        <AuthGuard>
          <div className="relative flex min-h-screen flex-col">
            <main className="flex-1">
              <Suspense fallback={
                <div className="container mx-auto max-w-7xl p-6">
                  <div className="animate-pulse bg-gray-200 rounded-lg h-32"></div>
                </div>
              }>
                {children}
              </Suspense>
            </main>
          </div>
        </AuthGuard>
      </body>
    </html>
  )
}
