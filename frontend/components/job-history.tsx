'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { 
  History, 
  Play, 
  Pause, 
  CheckCircle, 
  XCircle, 
  StopCircle, 
  Square,
  Trash2,
  RefreshCw,
  Filter,
  Clock,
  Calendar
} from 'lucide-react'
import { useJobs, useJobActions, useProjects } from '@/hooks/use-job-api'
import { Job, JobFilters } from '@/lib/api'

interface JobHistoryProps {
  refreshTrigger?: number
}

export function JobHistory({ refreshTrigger }: JobHistoryProps) {
  const router = useRouter()
  const [filters, setFilters] = useState<JobFilters>({ limit: 20 })
  const { projects } = useProjects()
  const { jobs, loading, error, refetch } = useJobs(filters)
  const { cancelJob, deleteJob, loading: actionLoading } = useJobActions()

  // Refresh when refreshTrigger changes
  useEffect(() => {
    if (refreshTrigger && refreshTrigger > 0) {
      refetch()
    }
  }, [refreshTrigger, refetch])

  const getStatusBadge = (status: Job['status']) => {
    const statusConfig = {
      pending: { color: 'bg-yellow-100 text-yellow-800', icon: Pause, label: '待機中' },
      running: { color: 'bg-blue-100 text-blue-800', icon: Play, label: '実行中' },
      completed: { color: 'bg-green-100 text-green-800', icon: CheckCircle, label: '完了' },
      failed: { color: 'bg-red-100 text-red-800', icon: XCircle, label: '失敗' },
      cancelled: { color: 'bg-gray-100 text-gray-800', icon: StopCircle, label: 'キャンセル' },
    }

    const config = statusConfig[status]
    const IconComponent = config.icon

    return (
      <Badge className={config.color}>
        <IconComponent className="mr-1 h-3 w-3" />
        {config.label}
      </Badge>
    )
  }

  const formatDateTime = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleString('ja-JP', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const getScheduleInfo = (job: Job) => {
    if (!job.schedule_type) {
      return null
    }

    const scheduleTypeLabel = {
      immediate: t('job.schedule.immediate'),
      after_reset: t('job.schedule.afterReset'),
      delayed: t('job.schedule.delayed'),
      scheduled: t('job.schedule.scheduled')
    }

    return {
      type: scheduleTypeLabel[job.schedule_type as keyof typeof scheduleTypeLabel] || job.schedule_type,
      scheduledAt: job.scheduled_at
    }
  }

  const formatDuration = (startedAt?: string, completedAt?: string) => {
    if (!startedAt) return '-'
    
    const start = new Date(startedAt)
    const end = completedAt ? new Date(completedAt) : new Date()
    const durationMs = end.getTime() - start.getTime()
    const seconds = Math.floor(durationMs / 1000)
    
    if (seconds < 60) return `${seconds}秒`
    const minutes = Math.floor(seconds / 60)
    if (minutes < 60) return `${minutes}分`
    const hours = Math.floor(minutes / 60)
    return `${hours}時間${minutes % 60}分`
  }

  const formatTimeUntilExecution = (scheduledAt: string) => {
    const scheduled = new Date(scheduledAt)
    const now = new Date()
    const diffMs = scheduled.getTime() - now.getTime()
    
    if (diffMs <= 0) return '実行待機中'
    
    const diffMinutes = Math.floor(diffMs / (1000 * 60))
    const diffHours = Math.floor(diffMinutes / 60)
    const diffDays = Math.floor(diffHours / 24)
    
    if (diffDays > 0) return `${diffDays}${t('job.daysAfter')}`
    if (diffHours > 0) return `${diffHours}${t('tokenUsage.hours')}${diffMinutes % 60}${t('job.minutesAfter')}`
    if (diffMinutes > 0) return `${diffMinutes}${t('job.minutesAfter')}`
    return '間もなく実行'
  }

  const handleCancel = async (jobId: string) => {
    try {
      await cancelJob(jobId)
      refetch()
    } catch {
      // Error handled by hook
    }
  }

  const handleDelete = async (jobId: string) => {
    if (!confirm(t('job.confirmDelete'))) return
    
    try {
      await deleteJob(jobId)
      refetch()
    } catch {
      // Error handled by hook
    }
  }

  const handleFilterChange = (key: keyof JobFilters, value: string | undefined) => {
    setFilters(prev => ({
      ...prev,
      [key]: value === 'all' ? undefined : value
    }))
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <History className="h-5 w-5" />
{t('job.history')}
            </CardTitle>
            <CardDescription>
              実行されたタスクの履歴と状態を確認できます。
            </CardDescription>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={refetch}
            disabled={loading}
          >
            <RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            更新
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {/* Filters */}
        <div className="flex gap-4 mb-4">
          <div className="flex items-center gap-2">
            <Filter className="h-4 w-4" />
            <Select
              value={filters.status || 'all'}
              onValueChange={(value) => handleFilterChange('status', value)}
            >
              <SelectTrigger className="w-32">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全ステータス</SelectItem>
                <SelectItem value="pending">待機中</SelectItem>
                <SelectItem value="running">実行中</SelectItem>
                <SelectItem value="completed">完了</SelectItem>
                <SelectItem value="failed">失敗</SelectItem>
                <SelectItem value="cancelled">キャンセル</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <Select
            value={filters.project_id || 'all'}
            onValueChange={(value) => handleFilterChange('project_id', value)}
          >
            <SelectTrigger className="w-48">
              <SelectValue placeholder="プロジェクト" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全プロジェクト</SelectItem>
              {projects.map((project) => (
                <SelectItem key={project.id} value={project.id}>
                  {project.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Error Display */}
        {error && (
          <Alert variant="destructive" className="mb-4">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {/* Jobs Table */}
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('session.status')}</TableHead>
                <TableHead>{t('session.project')}</TableHead>
                <TableHead>{t('job.command')}</TableHead>
                <TableHead>{t('job.schedule')}</TableHead>
                <TableHead>{t('job.duration')}</TableHead>
                <TableHead>{t('job.createdAt')}</TableHead>
                <TableHead className="text-right">{t('job.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8">
                    読み込み中...
                  </TableCell>
                </TableRow>
              ) : !jobs || jobs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
{t('job.notFound')}
                  </TableCell>
                </TableRow>
              ) : (
                jobs.map((job) => (
                  <TableRow 
                    key={job.id} 
                    className="cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() => router.push(`/jobs/${job.id}`)}
                  >
                    <TableCell>
                      {getStatusBadge(job.status)}
                    </TableCell>
                    <TableCell>
                      <div className="font-medium">{job.project?.name}</div>
                      <div className="text-sm text-muted-foreground">
                        {job.project?.path}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="max-w-xs truncate font-mono text-sm">
                        {job.command}
                      </div>
                      {job.yolo_mode && (
                        <Badge variant="outline" className="mt-1 text-xs">
                          YOLO
                        </Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      {(() => {
                        const scheduleInfo = getScheduleInfo(job)
                        if (!scheduleInfo) return '-'
                        
                        return (
                          <div className="flex flex-col gap-1">
                            <div className="flex items-center gap-1">
                              {job.schedule_type === 'immediate' && <Play className="h-3 w-3" />}
                              {job.schedule_type === 'after_reset' && <RefreshCw className="h-3 w-3" />}
                              {job.schedule_type === 'scheduled' && <Calendar className="h-3 w-3" />}
                              {job.schedule_type === 'delayed' && <Clock className="h-3 w-3" />}
                              <span className="text-sm">{scheduleInfo.type}</span>
                            </div>
                            {scheduleInfo.scheduledAt && (
                              <>
                                <div className={`text-xs ${
                                  job.status === 'pending' && (job.schedule_type === 'scheduled' || job.schedule_type === 'delayed')
                                    ? 'text-blue-600 font-medium' 
                                    : 'text-muted-foreground'
                                }`}>
                                  {job.status === 'pending' && (job.schedule_type === 'scheduled' || job.schedule_type === 'delayed') 
                                    ? `実行予定: ${formatDateTime(scheduleInfo.scheduledAt)}`
                                    : formatDateTime(scheduleInfo.scheduledAt)
                                  }
                                </div>
                                {job.status === 'pending' && (job.schedule_type === 'scheduled' || job.schedule_type === 'delayed') && (
                                  <div className="text-xs text-orange-600">
                                    {formatTimeUntilExecution(scheduleInfo.scheduledAt)}
                                  </div>
                                )}
                              </>
                            )}
                          </div>
                        )
                      })()}
                    </TableCell>
                    <TableCell>
                      {formatDuration(job.started_at, job.completed_at)}
                    </TableCell>
                    <TableCell>
                      {formatDateTime(job.created_at)}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        {job.status === 'running' && (
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={actionLoading}
                            onClick={(e) => {
                              e.stopPropagation()
                              handleCancel(job.id)
                            }}
                          >
                            <Square className="h-4 w-4" />
                          </Button>
                        )}
                        
                        {(job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') && (
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={actionLoading}
                            onClick={(e) => {
                              e.stopPropagation()
                              handleDelete(job.id)
                            }}
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination Info */}
        {jobs && jobs.length > 0 && (
          <div className="mt-4 text-sm text-muted-foreground">
            {jobs.length} {t('job.totalJobs')}
          </div>
        )}
      </CardContent>
    </Card>
  )
}