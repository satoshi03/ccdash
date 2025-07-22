"use client"

import { useState, useEffect } from 'react'
import { api, TokenUsage, Session } from '@/lib/api'

export function useTokenUsage() {
  const [data, setData] = useState<TokenUsage | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = async () => {
    try {
      setLoading(true)
      setError(null)
      const tokenUsage = await api.tokenUsage.getCurrent()
      setData(tokenUsage)
    } catch (err) {
      console.error('Error fetching token usage:', err)
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  return { data, loading, error, refetch: fetchData }
}

export function useSessions() {
  const [data, setData] = useState<Session[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = async () => {
    try {
      setLoading(true)
      setError(null)
      const result = await api.sessions.getAll()
      setData(result.sessions)
    } catch (err) {
      console.error('Error fetching sessions:', err)
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  return { data, loading, error, refetch: fetchData }
}

export function useAvailableTokens(plan: string = 'pro') {
  const [data, setData] = useState<{
    available_tokens: number
    plan: string
    usage_limit: number
    used_tokens: number
  } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = async () => {
    try {
      setLoading(true)
      setError(null)
      const result = await api.tokenUsage.getAvailable(plan.toLowerCase())
      setData(result)
    } catch (err) {
      console.error('Error fetching available tokens:', err)
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [plan])

  return { data, loading, error, refetch: fetchData }
}

export function useSyncLogs() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const sync = async () => {
    try {
      setLoading(true)
      setError(null)
      await api.sync.logs()
      return true
    } catch (err) {
      console.error('Error syncing logs:', err)
      setError(err instanceof Error ? err.message : 'Unknown error')
      return false
    } finally {
      setLoading(false)
    }
  }

  return { sync, loading, error }
}

