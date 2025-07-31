"use client"

import { useState, useEffect, useCallback, useRef } from 'react'
import { api, Job, JobStatus, ClaudeCommandRequest, AsyncCommandResponse } from '@/lib/api'

export interface UseAsyncJobOptions {
  pollingInterval?: number // ms, default 2000
  autoCleanup?: boolean    // Auto delete completed jobs, default true
  maxRetries?: number      // Max retry attempts for polling, default 10
}

export interface UseAsyncJobState {
  job: Job | null
  isPolling: boolean
  error: string | null
  retryCount: number
}

export function useAsyncJob(options: UseAsyncJobOptions = {}) {
  const {
    pollingInterval = 2000,
    autoCleanup = true,
    maxRetries = 10
  } = options

  const [state, setState] = useState<UseAsyncJobState>({
    job: null,
    isPolling: false,
    error: null,
    retryCount: 0
  })

  const pollingRef = useRef<NodeJS.Timeout | null>(null)
  const jobIdRef = useRef<string | null>(null)
  const retryCountRef = useRef(0)

  // Clean up polling on unmount
  useEffect(() => {
    return () => {
      if (pollingRef.current) {
        clearInterval(pollingRef.current)
      }
    }
  }, [])

  // Start polling for job status
  const startPolling = useCallback(async (jobId: string) => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current)
    }

    jobIdRef.current = jobId
    retryCountRef.current = 0

    setState(prev => ({
      ...prev,
      isPolling: true,
      error: null,
      retryCount: 0
    }))

    const poll = async () => {
      try {
        const job = await api.claude.getJob(jobId)
        
        setState(prev => ({
          ...prev,
          job,
          error: null,
          retryCount: retryCountRef.current
        }))

        // Stop polling if job is completed, failed, or cancelled
        if (job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') {
          if (pollingRef.current) {
            clearInterval(pollingRef.current)
            pollingRef.current = null
          }
          
          setState(prev => ({
            ...prev,
            isPolling: false
          }))

          // Auto cleanup completed jobs if enabled
          if (autoCleanup && job.status === 'completed') {
            setTimeout(async () => {
              try {
                await api.claude.deleteJob(jobId)
              } catch (error) {
                console.warn('Failed to auto-cleanup job:', error)
              }
            }, 5000) // Wait 5 seconds before cleanup
          }
        }

        retryCountRef.current = 0 // Reset retry count on success
      } catch (error) {
        retryCountRef.current++
        
        setState(prev => ({
          ...prev,
          error: error instanceof Error ? error.message : 'Unknown error',
          retryCount: retryCountRef.current
        }))

        // Stop polling if max retries exceeded
        if (retryCountRef.current >= maxRetries) {
          if (pollingRef.current) {
            clearInterval(pollingRef.current)
            pollingRef.current = null
          }
          
          setState(prev => ({
            ...prev,
            isPolling: false,
            error: `Max retries (${maxRetries}) exceeded for job polling`
          }))
        }
      }
    }

    // Initial poll
    await poll()

    // Start interval polling if job is still running
    if (state.job?.status === 'running' || state.job?.status === 'pending') {
      pollingRef.current = setInterval(poll, pollingInterval)
    }
  }, [pollingInterval, maxRetries, autoCleanup, state.job?.status])

  // Stop polling
  const stopPolling = useCallback(() => {
    if (pollingRef.current) {
      clearInterval(pollingRef.current)
      pollingRef.current = null
    }
    
    setState(prev => ({
      ...prev,
      isPolling: false
    }))
  }, [])

  // Execute command asynchronously and start polling
  const executeAsync = useCallback(async (request: ClaudeCommandRequest): Promise<string> => {
    try {
      setState(prev => ({
        ...prev,
        error: null
      }))

      const response: AsyncCommandResponse = await api.claude.executeCommandAsync(request)
      
      // Start polling for the job
      await startPolling(response.job_id)
      
      return response.job_id
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to execute command'
      }))
      throw error
    }
  }, [startPolling])

  // Cancel job
  const cancelJob = useCallback(async (jobId: string) => {
    try {
      await api.claude.cancelJob(jobId)
      
      // Update local state immediately
      setState(prev => prev.job && prev.job.id === jobId ? {
        ...prev,
        job: {
          ...prev.job,
          status: 'cancelled' as JobStatus,
          error: 'Job cancelled by user'
        }
      } : prev)
      
      stopPolling()
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to cancel job'
      }))
    }
  }, [stopPolling])

  // Delete job
  const deleteJob = useCallback(async (jobId: string) => {
    try {
      await api.claude.deleteJob(jobId)
      
      // Clear local state if this was the current job
      setState(prev => prev.job && prev.job.id === jobId ? {
        ...prev,
        job: null
      } : prev)
      
      stopPolling()
    } catch (error) {
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to delete job'
      }))
    }
  }, [stopPolling])

  // Get job by ID (one-time fetch)
  const getJob = useCallback(async (jobId: string): Promise<Job> => {
    const job = await api.claude.getJob(jobId)
    setState(prev => ({
      ...prev,
      job,
      error: null
    }))
    return job
  }, [])

  // Clear current job and stop polling
  const clearJob = useCallback(() => {
    stopPolling()
    setState({
      job: null,
      isPolling: false,
      error: null,
      retryCount: 0
    })
  }, [stopPolling])

  return {
    // State
    job: state.job,
    isPolling: state.isPolling,
    error: state.error,
    retryCount: state.retryCount,
    
    // Actions
    executeAsync,
    startPolling,
    stopPolling,
    cancelJob,
    deleteJob,
    getJob,
    clearJob,
    
    // Computed values
    isJobRunning: state.job?.status === 'running',
    isJobCompleted: state.job?.status === 'completed',
    isJobFailed: state.job?.status === 'failed',
    isJobCancelled: state.job?.status === 'cancelled',
    hasError: !!state.error
  }
}