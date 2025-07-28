"use client"

import { useState, useEffect, useCallback } from "react"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { P90ProgressCard } from "@/components/p90-progress-card"
import { SessionList } from "@/components/session-list"
import { ProjectOverview } from "@/components/project-overview"
import { Header } from "@/components/header"
import { useTokenUsage, useSessions, useSyncLogs, useP90Predictions } from "@/hooks/use-api"
import { useI18n } from "@/hooks/use-i18n"
import { Settings, getSettings } from "@/lib/settings"
import { Session } from "@/lib/api"

export default function Dashboard() {
  const { data: tokenUsage, loading: tokenLoading, error: tokenError, refetch: refetchTokens } = useTokenUsage()
  const { data: sessions, loading: sessionsLoading, error: sessionsError, refetch: refetchSessions } = useSessions()
  const { data: p90Predictions, loading: p90Loading, refetch: refetchP90 } = useP90Predictions()
  const { sync: syncLogs } = useSyncLogs()
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [settings, setSettings] = useState<Settings>(() => getSettings())
  
  const handleSettingsChange = (newSettings: Settings) => {
    setSettings(newSettings)
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

  const convertSessionsToProjects = (sessions: Session[]) => {
    const projectMap = new Map()
    
    sessions.forEach(session => {
      const projectPath = session.project_path
      const projectName = session.project_name
      
      if (!projectMap.has(projectPath)) {
        projectMap.set(projectPath, {
          id: projectPath, // プロジェクトパスを一意のIDとして使用
          name: projectName,
          originalPath: projectPath,
          sessions: []
        })
      }
      
      const project = projectMap.get(projectPath)
      project.sessions.push({
        id: `${session.id}-${session.start_time}`, // より一意性を高める
        sessionId: session.id,
        startTime: new Date(session.start_time),
        endTime: session.end_time ? new Date(session.end_time) : null,
        tokenUsage: session.total_tokens,
        status: session.is_active ? 'running' : 'completed',
        messageCount: session.message_count,
        codeGenerated: session.generated_code?.length > 0
      })
    })
    
    // プロジェクトを最終実行時間でソート
    const projectsArray = Array.from(projectMap.values())
    projectsArray.forEach(project => {
      // 各プロジェクト内のセッションを最終実行時間でソート
      project.sessions.sort((a: { endTime: Date | null; startTime: Date }, b: { endTime: Date | null; startTime: Date }) => {
        const aTime = a.endTime || a.startTime
        const bTime = b.endTime || b.startTime
        return bTime.getTime() - aTime.getTime()
      })
    })
    
    // プロジェクト自体も最終実行時間でソート
    projectsArray.sort((a, b) => {
      const aLastTime = a.sessions.length > 0 ? (a.sessions[0].endTime || a.sessions[0].startTime) : new Date(0)
      const bLastTime = b.sessions.length > 0 ? (b.sessions[0].endTime || b.sessions[0].startTime) : new Date(0)
      return bLastTime.getTime() - aLastTime.getTime()
    })
    
    return projectsArray
  }

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
        <Tabs defaultValue="overview" className="space-y-4">
          <TabsList>
            <TabsTrigger value="overview">{t('common.overview')}</TabsTrigger>
            <TabsTrigger value="sessions">{t('common.sessions')}</TabsTrigger>
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

        </Tabs>
      </div>
    </div>
  )
}