// Authentication Status Component
// Shows current auth status and provides logout functionality

'use client'

import React, { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { LogOut, Key, AlertCircle } from 'lucide-react'
import { apiClient } from '@/lib/api'

interface AuthStatusProps {
  className?: string
  showFullStatus?: boolean
}

export function AuthStatus({ className = '', showFullStatus = true }: AuthStatusProps) {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const checkAuth = () => {
      const hasKey = apiClient.hasApiKey()
      setIsAuthenticated(hasKey)
      setIsLoading(false)
    }

    checkAuth()
    
    // Check auth status periodically
    const interval = setInterval(checkAuth, 5000)
    return () => clearInterval(interval)
  }, [])

  const handleLogout = () => {
    apiClient.clearApiKey()
    setIsAuthenticated(false)
    // Optionally trigger a page reload to show auth form
    window.location.reload()
  }

  if (isLoading) {
    return (
      <div className={`flex items-center gap-2 ${className}`}>
        <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-500"></div>
        {showFullStatus && <span className="text-sm text-gray-500">認証確認中...</span>}
      </div>
    )
  }

  if (!isAuthenticated) {
    return (
      <div className={`flex items-center gap-2 ${className}`}>
        <Badge variant="destructive" className="bg-red-100 text-red-800">
          <AlertCircle className="w-3 h-3 mr-1" />
          未認証
        </Badge>
        {showFullStatus && (
          <span className="text-sm text-gray-500">APIキーが必要です</span>
        )}
      </div>
    )
  }

  return (
    <div className={`flex items-center gap-2 ${className}`}>
      <Badge variant="secondary" className="bg-green-100 text-green-800">
        <Key className="w-3 h-3 mr-1" />
        認証済み
      </Badge>
      {showFullStatus && (
        <Button
          onClick={handleLogout}
          variant="ghost"
          size="sm"
          className="h-8 px-2 text-xs"
        >
          <LogOut className="w-3 h-3 mr-1" />
          ログアウト
        </Button>
      )}
    </div>
  )
}