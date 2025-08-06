// API Key Authentication Component
// Allows users to securely input and manage their API key

'use client'

import React, { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Eye, EyeOff, Key, LogOut, Check } from 'lucide-react'
import { apiClient } from '@/lib/api'

interface ApiKeyAuthProps {
  onAuthStateChange?: (isAuthenticated: boolean) => void
}

export function ApiKeyAuth({ onAuthStateChange }: ApiKeyAuthProps) {
  const [apiKey, setApiKey] = useState('')
  const [showApiKey, setShowApiKey] = useState(false)
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [error, setError] = useState('')
  const [isValidating, setIsValidating] = useState(false)

  // Check authentication status on mount
  useEffect(() => {
    const hasKey = apiClient.hasApiKey()
    setIsAuthenticated(hasKey)
    onAuthStateChange?.(hasKey)
  }, [onAuthStateChange])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsValidating(true)

    if (!apiKey.trim()) {
      setError('APIキーを入力してください')
      setIsValidating(false)
      return
    }

    try {
      // Set the API key
      apiClient.setApiKey(apiKey.trim())

      // Test the API key by making a simple request
      await apiClient.getTokenUsage()

      // If successful, update state
      setIsAuthenticated(true)
      setApiKey('')
      onAuthStateChange?.(true)
    } catch {
      setError('無効なAPIキーです。正しいAPIキーを入力してください。')
      apiClient.clearApiKey()
      setIsAuthenticated(false)
      onAuthStateChange?.(false)
    } finally {
      setIsValidating(false)
    }
  }

  const handleLogout = () => {
    apiClient.clearApiKey()
    setIsAuthenticated(false)
    setApiKey('')
    setError('')
    onAuthStateChange?.(false)
  }

  if (isAuthenticated) {
    return (
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <div className="flex items-center justify-center gap-2">
            <Badge variant="secondary" className="bg-green-100 text-green-800">
              <Check className="w-3 h-3 mr-1" />
              認証済み
            </Badge>
          </div>
          <CardTitle className="text-lg">API認証</CardTitle>
          <CardDescription>
            APIキーが設定されています
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Button 
            onClick={handleLogout}
            variant="outline" 
            className="w-full"
          >
            <LogOut className="w-4 h-4 mr-2" />
            ログアウト
          </Button>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="w-full max-w-md">
      <CardHeader className="text-center">
        <div className="flex justify-center mb-2">
          <Key className="w-8 h-8 text-blue-500" />
        </div>
        <CardTitle className="text-xl">API認証</CardTitle>
        <CardDescription>
          CCDashにアクセスするためのAPIキーを入力してください
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <label htmlFor="apiKey" className="text-sm font-medium">
              APIキー
            </label>
            <div className="relative">
              <Input
                id="apiKey"
                type={showApiKey ? 'text' : 'password'}
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder="APIキーを入力してください"
                className="pr-10"
                disabled={isValidating}
              />
              <Button
                type="button"
                variant="ghost"
                size="sm"
                className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                onClick={() => setShowApiKey(!showApiKey)}
                disabled={isValidating}
              >
                {showApiKey ? (
                  <EyeOff className="w-4 h-4" />
                ) : (
                  <Eye className="w-4 h-4" />
                )}
              </Button>
            </div>
          </div>

          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <Button 
            type="submit" 
            className="w-full" 
            disabled={isValidating}
          >
            {isValidating ? '検証中...' : '認証'}
          </Button>
        </form>

        <div className="mt-6 pt-4 border-t text-center">
          <p className="text-xs text-gray-500">
            APIキーはブラウザのセッションストレージに安全に保存されます
          </p>
        </div>
      </CardContent>
    </Card>
  )
}