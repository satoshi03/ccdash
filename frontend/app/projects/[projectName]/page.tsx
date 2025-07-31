'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { ArrowLeft, Play, Clock, Calendar, Tokens } from 'lucide-react'
import Link from 'next/link'

interface SessionWindow {
  id: string
  project_name: string
  window_start: string
  window_end: string
  reset_time: string
  total_input_tokens: number
  total_output_tokens: number
  message_count: number
  duration_minutes: number
  status: string
}

interface ProjectStats {
  total_sessions: number
  total_input_tokens: number
  total_output_tokens: number
  total_messages: number
  avg_session_duration: number
  first_session: string
  last_session: string
}

export default function ProjectDetailPage() {
  const params = useParams()
  const projectName = params.projectName as string
  const decodedProjectName = decodeURIComponent(projectName)
  
  const [sessions, setSessions] = useState<SessionWindow[]>([])
  const [stats, setStats] = useState<ProjectStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchProjectData = async () => {
      try {
        setLoading(true)
        
        // セッション一覧を取得
        const sessionsResponse = await fetch('/api/claude/session-windows')
        if (!sessionsResponse.ok) {
          throw new Error('Failed to fetch sessions')
        }
        const allSessions: SessionWindow[] = await sessionsResponse.json()
        
        // プロジェクト名でフィルタリング
        const projectSessions = allSessions.filter(
          session => session.project_name === decodedProjectName
        )
        
        setSessions(projectSessions)
        
        // 統計情報を計算
        if (projectSessions.length > 0) {
          const totalInputTokens = projectSessions.reduce((sum, s) => sum + s.total_input_tokens, 0)
          const totalOutputTokens = projectSessions.reduce((sum, s) => sum + s.total_output_tokens, 0)
          const totalMessages = projectSessions.reduce((sum, s) => sum + s.message_count, 0)
          const avgDuration = projectSessions.reduce((sum, s) => sum + s.duration_minutes, 0) / projectSessions.length
          
          const sortedSessions = [...projectSessions].sort((a, b) => 
            new Date(a.window_start).getTime() - new Date(b.window_start).getTime()
          )
          
          setStats({
            total_sessions: projectSessions.length,
            total_input_tokens: totalInputTokens,
            total_output_tokens: totalOutputTokens,
            total_messages: totalMessages,
            avg_session_duration: Math.round(avgDuration),
            first_session: sortedSessions[0].window_start,
            last_session: sortedSessions[sortedSessions.length - 1].window_start
          })
        }
        
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error')
      } finally {
        setLoading(false)
      }
    }

    if (decodedProjectName) {
      fetchProjectData()
    }
  }, [decodedProjectName])

  const handleExecuteCommand = async (command: string) => {
    try {
      const response = await fetch('/api/claude/execute', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          command,
          project_name: decodedProjectName
        }),
      })
      
      if (!response.ok) {
        throw new Error('Failed to execute command')
      }
      
      const result = await response.json()
      console.log('Command executed:', result)
      // TODO: 結果表示のUIを実装
    } catch (err) {
      console.error('Error executing command:', err)
      // TODO: エラー表示のUIを実装
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading project data...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <p className="text-red-600 mb-4">Error: {error}</p>
          <Link href="/">
            <Button variant="outline">
              <ArrowLeft className="w-4 h-4 mr-2" />
              Back to Dashboard
            </Button>
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="container mx-auto px-4 py-8">
        {/* Header */}
        <div className="flex items-center gap-4 mb-8">
          <Link href="/">
            <Button variant="outline" size="sm">
              <ArrowLeft className="w-4 h-4 mr-2" />
              Back
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold text-gray-900">{decodedProjectName}</h1>
            <p className="text-gray-600">Project Details & Management</p>
          </div>
        </div>

        {/* Stats Cards */}
        {stats && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            <Card>
              <CardContent className="p-6">
                <div className="flex items-center">
                  <Calendar className="h-8 w-8 text-blue-600" />
                  <div className="ml-4">
                    <p className="text-sm font-medium text-gray-600">Total Sessions</p>
                    <p className="text-2xl font-bold text-gray-900">{stats.total_sessions}</p>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="p-6">
                <div className="flex items-center">
                  <Tokens className="h-8 w-8 text-green-600" />
                  <div className="ml-4">
                    <p className="text-sm font-medium text-gray-600">Total Tokens</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {(stats.total_input_tokens + stats.total_output_tokens).toLocaleString()}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="p-6">
                <div className="flex items-center">
                  <Clock className="h-8 w-8 text-orange-600" />
                  <div className="ml-4">
                    <p className="text-sm font-medium text-gray-600">Avg Duration</p>
                    <p className="text-2xl font-bold text-gray-900">{stats.avg_session_duration}m</p>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardContent className="p-6">
                <div className="flex items-center">
                  <div className="h-8 w-8 bg-purple-100 rounded-full flex items-center justify-center">
                    <span className="text-purple-600 font-semibold">#</span>
                  </div>
                  <div className="ml-4">
                    <p className="text-sm font-medium text-gray-600">Messages</p>
                    <p className="text-2xl font-bold text-gray-900">{stats.total_messages}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>
        )}

        {/* Tabs */}
        <Tabs defaultValue="sessions" className="space-y-6">
          <TabsList>
            <TabsTrigger value="sessions">Sessions</TabsTrigger>
            <TabsTrigger value="actions">Actions</TabsTrigger>
          </TabsList>

          <TabsContent value="sessions" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Session History</CardTitle>
              </CardHeader>
              <CardContent>
                {sessions.length === 0 ? (
                  <p className="text-gray-500 text-center py-8">No sessions found for this project.</p>
                ) : (
                  <div className="space-y-4">
                    {sessions.map((session) => (
                      <div key={session.id} className="border rounded-lg p-4 hover:bg-gray-50">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-4">
                            <div>
                              <p className="font-medium text-gray-900">
                                {new Date(session.window_start).toLocaleDateString()} 
                                {' '}
                                {new Date(session.window_start).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}
                              </p>
                              <p className="text-sm text-gray-600">
                                {session.message_count} messages • {session.duration_minutes}m
                              </p>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            <Badge variant={session.status === 'active' ? 'default' : 'secondary'}>
                              {session.status}
                            </Badge>
                            <div className="text-right">
                              <p className="text-sm font-medium">
                                {(session.total_input_tokens + session.total_output_tokens).toLocaleString()} tokens
                              </p>
                              <p className="text-xs text-gray-600">
                                {session.total_input_tokens.toLocaleString()} in / {session.total_output_tokens.toLocaleString()} out
                              </p>
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="actions" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Project Actions</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Button 
                    onClick={() => handleExecuteCommand('npm run dev')}
                    className="flex items-center gap-2"
                  >
                    <Play className="w-4 h-4" />
                    Start Dev Server
                  </Button>
                  
                  <Button 
                    onClick={() => handleExecuteCommand('npm run build')}
                    variant="outline"
                    className="flex items-center gap-2"
                  >
                    <Play className="w-4 h-4" />
                    Build Project
                  </Button>
                  
                  <Button 
                    onClick={() => handleExecuteCommand('npm run test')}
                    variant="outline"
                    className="flex items-center gap-2"
                  >
                    <Play className="w-4 h-4" />
                    Run Tests
                  </Button>
                  
                  <Button 
                    onClick={() => handleExecuteCommand('npm run lint')}
                    variant="outline"
                    className="flex items-center gap-2"
                  >
                    <Play className="w-4 h-4" />
                    Run Lint
                  </Button>
                </div>
                
                <div className="pt-4 border-t">
                  <p className="text-sm text-gray-600 mb-2">
                    Custom commands will be available in future versions.
                  </p>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}