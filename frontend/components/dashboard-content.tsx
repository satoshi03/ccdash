"use client"

import { useState, useEffect, useCallback } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { P90ProgressCard } from "@/components/p90-progress-card"
import { SessionList } from "@/components/session-list"
import { ProjectOverview } from "@/components/project-overview"
import { TaskExecutionForm } from "@/components/task-execution-form"
import { JobHistory } from "@/components/job-history"
import { Header } from "@/components/header"
import { useTokenUsage, useSessions, useSyncLogs, useP90Predictions } from "@/hooks/use-api"
import { useI18n } from "@/hooks/use-i18n"
import { Settings, getSettings } from "@/lib/settings"
import { convertSessionsToProjects } from "@/lib/project-utils"

export default function Dashboard() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { data: tokenUsage, loading: tokenLoading, error: tokenError, refetch: refetchTokens } = useTokenUsage()
  const { data: sessions, loading: sessionsLoading, error: sessionsError, refetch: refetchSessions } = useSessions()
  const { data: p90Predictions, loading: p90Loading, refetch: refetchP90 } = useP90Predictions()
  const { sync: syncLogs } = useSyncLogs()
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [settings, setSettings] = useState<Settings>(() => getSettings())
  
  // Task execution states
  const [refreshTrigger, setRefreshTrigger] = useState(0)
  
  // Tab state from URL
  const currentTab = searchParams.get('tab') || 'overview'
  
  const handleSettingsChange = (newSettings: Settings) => {
    setSettings(newSettings)
  }
  
  // Task execution event handlers
  const handleJobCreated = () => {
    setRefreshTrigger(prev => prev + 1)
  }
  
  const handleTabChange = (value: string) => {
    const params = new URLSearchParams(searchParams.toString())
    params.set('tab', value)
    router.push(`/?${params.toString()}`)
  }
  
  
  const { t } = useI18n()

  const refreshData = useCallback(async () => {
    setIsRefreshing(true)
    try {
      // First sync logs to ensure database is up to date
      await syncLogs()
      // Then fetch the updated data
      await Promise.all([
        refetchTokens(),
        refetchSessions(),
        refetchP90()
      ])
    } catch (error) {
      console.error('Error refreshing data:', error)
    } finally {
      setIsRefreshing(false)
    }
  }, [syncLogs, refetchTokens, refetchSessions, refetchP90])

  // 自動更新機能（設定可能な間隔）
  useEffect(() => {
    const interval = setInterval(() => {
      if (!isRefreshing) {
        refreshData()
      }
    }, settings.autoRefreshInterval * 1000) // 設定値を秒からミリ秒に変換

    return () => clearInterval(interval)
  }, [isRefreshing, settings.autoRefreshInterval, refreshData])



  const projects = sessions ? convertSessionsToProjects(sessions) : []
  const resetTime = tokenUsage ? new Date(tokenUsage.window_end) : new Date(Date.now() + 5 * 60 * 60 * 1000)

  return (
    <div className="min-h-screen bg-background">
      <Header onSettingsChange={handleSettingsChange} />
      <div className="container mx-auto max-w-7xl p-6 space-y-6">

        {/* Token Usage Overview */}
        {tokenLoading ? (
          <div className="animate-pulse bg-gray-200 rounded-lg h-32"></div>
        ) : tokenError ? (
          <div className="bg-red-50 border border-red-200 rounded-lg p-4">
            <p className="text-red-600">{t('errors.tokenUsageFetch')}: {tokenError}</p>
          </div>
        ) : tokenUsage ? (
          <P90ProgressCard
            currentTokens={tokenUsage.total_tokens}
            currentMessages={tokenUsage.total_messages}
            currentCost={tokenUsage.total_cost}
            p90Prediction={p90Predictions}
            resetTime={resetTime}
            isLoading={p90Loading}
            settings={settings}
          />
        ) : null}

        {/* Main Content */}
        <Tabs value={currentTab} onValueChange={handleTabChange} className="space-y-4">
          <TabsList>
            <TabsTrigger value="overview">{t('common.overview')}</TabsTrigger>
            <TabsTrigger value="sessions">{t('common.sessions')}</TabsTrigger>
            <TabsTrigger value="tasks">{t('job.execution')}</TabsTrigger>
          </TabsList>

          <TabsContent value="overview" className="space-y-4">
            {sessionsLoading ? (
              <div className="animate-pulse bg-gray-200 rounded-lg h-64"></div>
            ) : sessionsError ? (
              <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                <p className="text-red-600">{t('errors.sessionsFetch')}: {sessionsError}</p>
              </div>
            ) : (
              <ProjectOverview projects={projects} />
            )}
          </TabsContent>

          <TabsContent value="sessions" className="space-y-4">
            {sessionsLoading ? (
              <div className="animate-pulse bg-gray-200 rounded-lg h-64"></div>
            ) : sessionsError ? (
              <div className="bg-red-50 border border-red-200 rounded-lg p-4">
                <p className="text-red-600">{t('errors.sessionsFetch')}: {sessionsError}</p>
              </div>
            ) : (
              <SessionList projects={projects} />
            )}
          </TabsContent>

          <TabsContent value="tasks" className="space-y-6">
            {/* Task Execution Form */}
            <TaskExecutionForm 
              onJobCreated={handleJobCreated} 
            />
            
            {/* Job History */}
            <JobHistory 
              refreshTrigger={refreshTrigger}
            />
          </TabsContent>

        </Tabs>
      </div>
    </div>
  )
}