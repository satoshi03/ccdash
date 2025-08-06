// Authentication Guard Component
// Protects components by requiring API key authentication

'use client'

import React, { useState, useEffect } from 'react'
import { apiClient } from '@/lib/api'
import { ApiKeyAuth } from './ApiKeyAuth'

interface AuthGuardProps {
  children: React.ReactNode
  fallback?: React.ReactNode
}

export function AuthGuard({ children, fallback }: AuthGuardProps) {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    // Check if we have a valid API key
    const checkAuth = async () => {
      try {
        if (apiClient.hasApiKey()) {
          // Test the API key by making a request
          await apiClient.getTokenUsage()
          setIsAuthenticated(true)
        } else {
          setIsAuthenticated(false)
        }
      } catch {
        // API key is invalid, clear it
        apiClient.clearApiKey()
        setIsAuthenticated(false)
      } finally {
        setIsLoading(false)
      }
    }

    checkAuth()
  }, [])

  const handleAuthStateChange = (authenticated: boolean) => {
    setIsAuthenticated(authenticated)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500 mx-auto mb-4"></div>
          <p className="text-sm text-gray-500">認証状態を確認中...</p>
        </div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-gray-50">
        <div className="w-full max-w-md px-4">
          {fallback || (
            <ApiKeyAuth onAuthStateChange={handleAuthStateChange} />
          )}
        </div>
      </div>
    )
  }

  return <>{children}</>
}