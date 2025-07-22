import type { Metadata } from 'next'
import { Suspense } from 'react'
import './globals.css'
import { Header } from '@/components/header'

export const metadata: Metadata = {
  title: 'Claudeee - Claude Code Monitor & Task Scheduler',
  description: 'Monitor Claude Code usage, manage sessions, and schedule tasks efficiently. Track token consumption, session activities, and optimize your AI development workflow.',
  keywords: ['Claude Code', 'AI Development', 'Token Monitoring', 'Task Scheduler', 'Development Tools'],
  generator: 'React + Next.js + TailwindCSS + TypeScript + Go',
  authors: [{ name: 'Claudeee Team' }],
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
    <html lang="en" suppressHydrationWarnings>
      <body className="min-h-screen bg-background font-sans antialiased">
        <div className="relative flex min-h-screen flex-col">
          <Suspense fallback={
            <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
              <div className="container mx-auto max-w-7xl flex h-16 items-center px-6">
                <div className="flex items-center space-x-2">
                  <div className="h-8 w-8 rounded-lg bg-gray-200 animate-pulse"></div>
                  <div className="h-5 w-24 rounded bg-gray-200 animate-pulse"></div>
                </div>
              </div>
            </header>
          }>
            <Header />
          </Suspense>
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
      </body>
    </html>
  )
}
