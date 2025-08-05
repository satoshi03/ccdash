"use client"

import { useState, useEffect } from 'react'
import { api, TokenUsage, Session, P90Prediction, BurnRatePoint } from '@/lib/api'

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
      setError(err instanceof Error ? err.message : 'Unknown error')
      return false
    } finally {
      setLoading(false)
    }
  }

  return { sync, loading, error }
}

export function useP90Predictions() {
  const [data, setData] = useState<P90Prediction | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = async () => {
    try {
      setLoading(true)
      setError(null)
      const prediction = await api.predictions.getP90()
      setData(prediction)
    } catch (err) {
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

export function useP90PredictionsByProject(projectName: string) {
  const [data, setData] = useState<P90Prediction | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = async () => {
    if (!projectName) return
    
    try {
      setLoading(true)
      setError(null)
      const prediction = await api.predictions.getP90ByProject(projectName)
      setData(prediction)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [projectName])

  return { data, loading, error, refetch: fetchData }
}

export function useBurnRateHistory(hours: number = 24) {
  const [data, setData] = useState<BurnRatePoint[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = async () => {
    try {
      setLoading(true)
      setError(null)
      const result = await api.predictions.getBurnRateHistory(hours)
      setData(result.burn_rate_history)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [hours])

  return { data, loading, error, refetch: fetchData }
}

