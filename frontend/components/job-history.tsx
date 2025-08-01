'use client'

import { useState } from 'react'
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
  Eye, 
  Square,
  Trash2,
  RefreshCw,
  Filter
} from 'lucide-react'
import { useJobs, useJobActions, useProjects } from '@/hooks/use-job-api'
import { Job, JobFilters } from '@/lib/api'

interface JobHistoryProps {
  onJobSelect?: (job: Job) => void
  refreshTrigger?: number
}

export function JobHistory({ onJobSelect, refreshTrigger }: JobHistoryProps) {
  const [filters, setFilters] = useState<JobFilters>({ limit: 20 })
  const { projects } = useProjects()
  const { jobs, loading, error, refetch } = useJobs(filters)
  const { cancelJob, deleteJob, loading: actionLoading } = useJobActions()

  // Refresh when refreshTrigger changes
  useState(() => {
    if (refreshTrigger) {
      refetch()
    }
  })

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

  const handleCancel = async (jobId: string) => {
    try {
      await cancelJob(jobId)
      refetch()
    } catch {
      // Error handled by hook
    }
  }

  const handleDelete = async (jobId: string) => {
    if (!confirm('このジョブを削除しますか？')) return
    
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
              ジョブ履歴
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
                <TableHead>ステータス</TableHead>
                <TableHead>プロジェクト</TableHead>
                <TableHead>コマンド</TableHead>
                <TableHead>実行時間</TableHead>
                <TableHead>作成日時</TableHead>
                <TableHead className="text-right">アクション</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8">
                    読み込み中...
                  </TableCell>
                </TableRow>
              ) : jobs.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-8 text-muted-foreground">
                    ジョブが見つかりません
                  </TableCell>
                </TableRow>
              ) : (
                jobs.map((job) => (
                  <TableRow key={job.id}>
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
                      {formatDuration(job.started_at, job.completed_at)}
                    </TableCell>
                    <TableCell>
                      {formatDateTime(job.created_at)}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => onJobSelect?.(job)}
                        >
                          <Eye className="h-4 w-4" />
                        </Button>
                        
                        {job.status === 'running' && (
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={actionLoading}
                            onClick={() => handleCancel(job.id)}
                          >
                            <Square className="h-4 w-4" />
                          </Button>
                        )}
                        
                        {(job.status === 'completed' || job.status === 'failed' || job.status === 'cancelled') && (
                          <Button
                            variant="ghost"
                            size="sm"
                            disabled={actionLoading}
                            onClick={() => handleDelete(job.id)}
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
        {jobs.length > 0 && (
          <div className="mt-4 text-sm text-muted-foreground">
            {jobs.length} 件のジョブを表示
          </div>
        )}
      </CardContent>
    </Card>
  )
}