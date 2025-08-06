// Standalone Authentication Page
// Can be used for explicit login or API key management

'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { ApiKeyAuth } from '@/components/auth'

export default function AuthPage() {
  const router = useRouter()
  const [isAuthenticating, setIsAuthenticating] = useState(false)

  const handleAuthSuccess = (authenticated: boolean) => {
    if (authenticated) {
      setIsAuthenticating(true)
      // Redirect to dashboard after successful authentication
      setTimeout(() => {
        router.push('/')
      }, 1000)
    }
  }

  if (isAuthenticating) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gray-50">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto mb-4"></div>
          <p className="text-sm text-gray-500">ダッシュボードにリダイレクト中...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-50">
      <div className="w-full max-w-md px-4">
        <ApiKeyAuth onAuthStateChange={handleAuthSuccess} />
      </div>
    </div>
  )
}