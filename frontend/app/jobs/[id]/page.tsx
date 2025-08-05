"use client"

import { useState, useEffect, Suspense } from "react"
import { useParams, useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  ArrowLeft,
  Play,
  Pause,
  CheckCircle,
  XCircle,
  StopCircle,
  Copy,
  Check,
  RefreshCw,
  Terminal,
  FileText,
  Clock,
  Folder
} from "lucide-react"
import { useJob } from "@/hooks/use-job-api"
import { Job } from "@/lib/api"
import { Header } from "@/components/header"

function JobDetailContent() {
  const params = useParams()
  const router = useRouter()
  const jobId = params.id as string

  const { job, loading, error, isRunning, refetch } = useJob(jobId)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [copiedField, setCopiedField] = useState<string | null>(null)

  // Auto-refresh for running jobs
  useEffect(() => {
    if (!jobId || !autoRefresh) return

    const interval = setInterval(() => {
      if (job?.status === 'running') {
        refetch()
      }
    }, 2000) // Refresh every 2 seconds for running jobs

    return () => clearInterval(interval)
  }, [jobId, job?.status, autoRefresh, refetch])

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
      second: '2-digit',
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
    if (minutes < 60) return `${minutes}分${seconds % 60}秒`
    const hours = Math.floor(minutes / 60)
    return `${hours}時間${minutes % 60}分`
  }

  const copyToClipboard = async (text: string, fieldName: string) => {
    try {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        await navigator.clipboard.writeText(text)
      } else {
        // Fallback for older browsers or non-HTTPS contexts
        const textArea = document.createElement('textarea')
        textArea.value = text
        document.body.appendChild(textArea)
        textArea.select()
        document.execCommand('copy')
        document.body.removeChild(textArea)
      }
      setCopiedField(fieldName)
      setTimeout(() => setCopiedField(null), 2000)
    } catch (error) {
      console.error('Failed to copy to clipboard:', error)
    }
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
          <p>ジョブ詳細を読み込み中...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <p className="text-red-600 mb-4">エラー: {error}</p>
          <Button onClick={() => router.push('/')}>戻る</Button>
        </div>
      </div>
    )
  }

  if (!job) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <p className="mb-4">ジョブが見つかりません</p>
          <Button onClick={() => router.push('/')}>戻る</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background">
      <Header />
      <div className="container mx-auto max-w-7xl p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" onClick={() => router.push('/?tab=tasks')}>
            <ArrowLeft className="w-4 h-4 mr-2" />
            戻る
          </Button>
          <div>
            <h1 className="text-2xl font-bold">ジョブ詳細</h1>
            <p className="text-muted-foreground">
              ジョブID: {jobId}
            </p>
          </div>
        </div>

        {/* Job Status and Controls */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            {getStatusBadge(job.status)}
            {job.yolo_mode && (
              <Badge variant="outline">YOLO Mode</Badge>
            )}
            {isRunning && (
              <Badge variant="outline" className="text-blue-600">
                <Play className="mr-1 h-3 w-3" />
                実行中
              </Badge>
            )}
          </div>
          
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={refetch}
              disabled={loading}
            >
              <RefreshCw className={`mr-1 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              更新
            </Button>
            
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
                className="rounded"
              />
              自動更新
            </label>
          </div>
        </div>

        {/* Job Info Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm flex items-center gap-2">
                <Folder className="h-4 w-4" />
                プロジェクト情報
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <div>
                <div className="text-sm font-medium">{job.project?.name}</div>
                <div className="text-xs text-muted-foreground">{job.project?.path}</div>
              </div>
              <div>
                <div className="text-xs text-muted-foreground">実行ディレクトリ</div>
                <div className="text-sm font-mono">{job.execution_directory}</div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-sm flex items-center gap-2">
                <Clock className="h-4 w-4" />
                実行情報
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <div>
                <div className="text-xs text-muted-foreground">作成日時</div>
                <div className="text-sm">{formatDateTime(job.created_at)}</div>
              </div>
              {job.started_at && (
                <div>
                  <div className="text-xs text-muted-foreground">開始日時</div>
                  <div className="text-sm">{formatDateTime(job.started_at)}</div>
                </div>
              )}
              {job.completed_at && (
                <div>
                  <div className="text-xs text-muted-foreground">完了日時</div>
                  <div className="text-sm">{formatDateTime(job.completed_at)}</div>
                </div>
              )}
              <div>
                <div className="text-xs text-muted-foreground">実行時間</div>
                <div className="text-sm">{formatDuration(job.started_at, job.completed_at)}</div>
              </div>
              {job.pid && (
                <div>
                  <div className="text-xs text-muted-foreground">プロセスID</div>
                  <div className="text-sm font-mono">{job.pid}</div>
                </div>
              )}
              {job.exit_code !== undefined && (
                <div>
                  <div className="text-xs text-muted-foreground">終了コード</div>
                  <div className="text-sm font-mono">{job.exit_code}</div>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Command */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm flex items-center gap-2">
                <Terminal className="h-4 w-4" />
                コマンド
              </CardTitle>
              <Button
                variant="outline"
                size="sm"
                onClick={() => copyToClipboard(job.command, 'command')}
                className="h-6 w-6 p-0"
              >
                {copiedField === 'command' ? (
                  <Check className="w-3 h-3" />
                ) : (
                  <Copy className="w-3 h-3" />
                )}
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="bg-gray-50 dark:bg-gray-900 p-3 rounded-md font-mono text-sm">
              {job.command}
            </div>
          </CardContent>
        </Card>

        {/* Logs */}
        <Tabs defaultValue="output" className="w-full">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="output" className="flex items-center gap-2">
              <FileText className="h-4 w-4" />
              出力ログ
            </TabsTrigger>
            <TabsTrigger value="error" className="flex items-center gap-2">
              <XCircle className="h-4 w-4" />
              エラーログ
            </TabsTrigger>
          </TabsList>
          
          <TabsContent value="output" className="mt-4">
            <Card>
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm">標準出力</CardTitle>
                  {job.output_log && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => copyToClipboard(job.output_log || '', 'output')}
                      className="h-6 w-6 p-0"
                    >
                      {copiedField === 'output' ? (
                        <Check className="w-3 h-3" />
                      ) : (
                        <Copy className="w-3 h-3" />
                      )}
                    </Button>
                  )}
                </div>
              </CardHeader>
              <CardContent>
                <ScrollArea className="h-96 w-full rounded-md border">
                  <div className="p-4">
                    {job.output_log ? (
                      <pre className="text-sm font-mono whitespace-pre-wrap">
                        {job.output_log}
                      </pre>
                    ) : (
                      <div className="text-sm text-muted-foreground">
                        出力ログがありません
                      </div>
                    )}
                  </div>
                </ScrollArea>
              </CardContent>
            </Card>
          </TabsContent>
          
          <TabsContent value="error" className="mt-4">
            <Card>
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-sm">標準エラー出力</CardTitle>
                  {job.error_log && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => copyToClipboard(job.error_log || '', 'error')}
                      className="h-6 w-6 p-0"
                    >
                      {copiedField === 'error' ? (
                        <Check className="w-3 h-3" />
                      ) : (
                        <Copy className="w-3 h-3" />
                      )}
                    </Button>
                  )}
                </div>
              </CardHeader>
              <CardContent>
                <ScrollArea className="h-96 w-full rounded-md border">
                  <div className="p-4">
                    {job.error_log ? (
                      <pre className="text-sm font-mono whitespace-pre-wrap text-red-600">
                        {job.error_log}
                      </pre>
                    ) : (
                      <div className="text-sm text-muted-foreground">
                        エラーログがありません
                      </div>
                    )}
                  </div>
                </ScrollArea>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}

function LoadingFallback() {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className="text-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
        <p>ジョブ詳細を読み込み中...</p>
      </div>
    </div>
  )
}

export default function JobDetailPage() {
  return (
    <Suspense fallback={<LoadingFallback />}>
      <JobDetailContent />
    </Suspense>
  )
}