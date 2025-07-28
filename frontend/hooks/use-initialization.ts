'use client'

import { useState, useEffect, useCallback } from 'react'
import { api, InitializationStatus } from '@/lib/api'

export function useInitialization() {
  const [status, setStatus] = useState<InitializationStatus | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const checkStatus = useCallback(async () => {
    try {
      setError(null)
      const result = await api.initialization.getStatus()
      setStatus(result)
      
      // Continue polling if still initializing
      if (result.status === 'initializing') {
        setTimeout(() => checkStatus(), 2000) // Poll every 2 seconds
      }
    } catch (err) {
      console.error('Failed to check initialization status:', err)
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    checkStatus()
  }, [checkStatus])

  const retry = useCallback(() => {
    setIsLoading(true)
    setError(null)
    checkStatus()
  }, [checkStatus])

  return {
    status,
    isLoading,
    error,
    retry,
    isInitializing: status?.status === 'initializing',
    isCompleted: status?.status === 'completed',
    isFailed: status?.status === 'failed',
  }
}