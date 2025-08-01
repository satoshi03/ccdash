'use client'

import { useState, useEffect, useCallback } from 'react'
import { api, Job, CreateJobRequest, JobFilters, Project } from '@/lib/api'

export function useJobs(filters?: JobFilters) {
  const [jobs, setJobs] = useState<Job[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [count, setCount] = useState(0)

  const fetchJobs = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const response = await api.jobs.getAll(filters)
      setJobs(response.jobs)
      setCount(response.count)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch jobs')
      setJobs([])
      setCount(0)
    } finally {
      setLoading(false)
    }
  }, [filters])

  useEffect(() => {
    fetchJobs()
  }, [fetchJobs])

  return {
    jobs,
    loading,
    error,
    count,
    refetch: fetchJobs,
  }
}

export function useJob(id: string | null) {
  const [job, setJob] = useState<Job | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isRunning, setIsRunning] = useState(false)

  const fetchJob = useCallback(async () => {
    if (!id) {
      setJob(null)
      setLoading(false)
      return
    }

    try {
      setLoading(true)
      setError(null)
      const response = await api.jobs.getById(id)
      setJob(response.job)
      setIsRunning(response.is_running)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch job')
      setJob(null)
      setIsRunning(false)
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    fetchJob()
  }, [fetchJob])

  return {
    job,
    loading,
    error,
    isRunning,
    refetch: fetchJob,
  }
}

export function useCreateJob() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const createJob = useCallback(async (request: CreateJobRequest) => {
    try {
      setLoading(true)
      setError(null)
      const response = await api.jobs.create(request)
      return response.job
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to create job'
      setError(errorMessage)
      throw new Error(errorMessage)
    } finally {
      setLoading(false)
    }
  }, [])

  return {
    createJob,
    loading,
    error,
  }
}

export function useJobActions() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const cancelJob = useCallback(async (id: string) => {
    try {
      setLoading(true)
      setError(null)
      await api.jobs.cancel(id)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to cancel job'
      setError(errorMessage)
      throw new Error(errorMessage)
    } finally {
      setLoading(false)
    }
  }, [])

  const deleteJob = useCallback(async (id: string) => {
    try {
      setLoading(true)
      setError(null)
      await api.jobs.delete(id)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to delete job'
      setError(errorMessage)
      throw new Error(errorMessage)
    } finally {
      setLoading(false)
    }
  }, [])

  return {
    cancelJob,
    deleteJob,
    loading,
    error,
  }
}

export function useProjects() {
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchProjects = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const response = await api.projects.getAll()
      setProjects(response.projects)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch projects')
      setProjects([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchProjects()
  }, [fetchProjects])

  return {
    projects,
    loading,
    error,
    refetch: fetchProjects,
  }
}

export function useJobQueueStatus() {
  const [status, setStatus] = useState<{
    running_jobs: number
    queued_jobs: number
    worker_count: number
    claude_available: boolean
  } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchStatus = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)
      const response = await api.jobs.getQueueStatus()
      setStatus(response)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch queue status')
      setStatus(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchStatus()
    const interval = setInterval(fetchStatus, 5000) // Update every 5 seconds
    return () => clearInterval(interval)
  }, [fetchStatus])

  return {
    status,
    loading,
    error,
    refetch: fetchStatus,
  }
}