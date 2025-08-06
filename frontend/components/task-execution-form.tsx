'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Loader2, Play, AlertCircle, Clock, Info } from 'lucide-react'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { useCreateJob, useProjects } from '@/hooks/use-job-api'
import { useI18n } from '@/hooks/use-i18n'
import { CreateJobRequest } from '@/lib/api'
import { Slider } from '@/components/ui/slider'
import { Input } from '@/components/ui/input'
import { format } from 'date-fns'
import { ja } from 'date-fns/locale'
interface TaskExecutionFormProps {
  onJobCreated?: () => void
}

export function TaskExecutionForm({ onJobCreated }: TaskExecutionFormProps) {
  const { createJob, loading: createLoading, error: createError } = useCreateJob()
  const { projects, loading: projectsLoading, error: projectsError } = useProjects()
  const { t } = useI18n()
  
  const [selectedProjectId, setSelectedProjectId] = useState<string>('')
  const [command, setCommand] = useState('')
  const [yoloMode, setYoloMode] = useState(false)
  const [scheduleType, setScheduleType] = useState('immediate')
  const [delayHours, setDelayHours] = useState(1)
  const [scheduledDate, setScheduledDate] = useState('')
  const [scheduledTime, setScheduledTime] = useState('')
  const [success, setSuccess] = useState<string | null>(null)
  const [validationError, setValidationError] = useState<string | null>(null)

  const validateScheduledDateTime = (): boolean => {
    if (scheduleType !== 'scheduled') return true
    
    if (!scheduledDate || !scheduledTime) {
      setValidationError('日付と時刻を両方指定してください')
      return false
    }
    
    const scheduledDateTime = new Date(`${scheduledDate}T${scheduledTime}:00`)
    const now = new Date()
    
    if (scheduledDateTime <= now) {
      setValidationError(t('job.validation.futureTime'))
      return false
    }
    
    // 1年以上先の日時は許可しない
    const oneYearFromNow = new Date()
    oneYearFromNow.setFullYear(oneYearFromNow.getFullYear() + 1)
    if (scheduledDateTime > oneYearFromNow) {
      setValidationError(t('job.validation.withinYear'))
      return false
    }
    
    return true
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!selectedProjectId || !command.trim()) {
      return
    }

    // Clear previous errors
    setValidationError(null)
    
    // Validate scheduled date/time
    if (!validateScheduledDateTime()) {
      return
    }

    try {
      setSuccess(null)
      const request: CreateJobRequest = {
        project_id: selectedProjectId,
        command: command.trim(),
        yolo_mode: yoloMode,
        schedule_type: scheduleType,
      }

      // Add schedule params based on schedule type
      if (scheduleType === 'delayed') {
        request.schedule_params = {
          delay_hours: delayHours
        }
      } else if (scheduleType === 'scheduled' && scheduledDate && scheduledTime) {
        const scheduledDateTime = new Date(`${scheduledDate}T${scheduledTime}:00`)
        request.schedule_params = {
          scheduled_time: scheduledDateTime.toISOString()
        }
      }

      const job = await createJob(request)
      setSuccess(`${t('job.created')}: ${job.id}`)
      
      // Reset form
      setCommand('')
      setYoloMode(false)
      setScheduleType('immediate')
      setDelayHours(1)
      setScheduledDate('')
      setScheduledTime('')
      setValidationError(null)
      
      // Notify parent component
      if (onJobCreated) {
        onJobCreated()
      }
    } catch {
      // Error is handled by the hook
    }
  }

  const selectedProject = projects?.find(p => p.id === selectedProjectId)

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Play className="h-5 w-5" />
{t('job.execution')}
        </CardTitle>
        <CardDescription>
          {t('job.description')}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Project Selection */}
          <div className="space-y-2">
            <Label htmlFor="project">{t('session.project')}</Label>
            <Select value={selectedProjectId} onValueChange={setSelectedProjectId}>
              <SelectTrigger>
                <SelectValue placeholder="プロジェクトを選択してください" />
              </SelectTrigger>
              <SelectContent>
                {projectsLoading ? (
                  <SelectItem value="loading" disabled>
                    読み込み中...
                  </SelectItem>
                ) : projectsError ? (
                  <SelectItem value="error" disabled>
                    エラー: {projectsError}
                  </SelectItem>
                ) : !projects || projects.length === 0 ? (
                  <SelectItem value="no-projects" disabled>
                    プロジェクトがありません
                  </SelectItem>
                ) : (
                  projects.map((project) => (
                    <SelectItem key={project.id} value={project.id}>
                      {project.name}
                    </SelectItem>
                  ))
                )}
              </SelectContent>
            </Select>
            {selectedProject && (
              <div className="text-sm text-muted-foreground">
                実行ディレクトリ: {selectedProject.path}
              </div>
            )}
          </div>

          {/* Command Input */}
          <div className="space-y-2">
            <Label htmlFor="command">{t('job.command')}</Label>
            <Textarea
              id="command"
              placeholder="例: 新しい機能を実装して..."
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              rows={6}
              className="resize-none"
            />
            <div className="text-sm text-muted-foreground">
              Claude Codeに実行させたいタスクを自然言語で記述してください。
            </div>
          </div>

          {/* YOLO Mode */}
          <div className="flex items-center space-x-2">
            <Checkbox
              id="yolo-mode"
              checked={yoloMode}
              onCheckedChange={(checked) => setYoloMode(checked as boolean)}
            />
            <Label htmlFor="yolo-mode" className="text-sm font-medium">
              YOLOモード
            </Label>
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Info className="h-4 w-4 text-amber-500 cursor-help hover:text-amber-600" />
                </TooltipTrigger>
                <TooltipContent className="max-w-md p-4">
                  <div className="space-y-2">
                    <p className="font-semibold text-amber-600">⚠️ セキュリティ警告</p>
                    <p className="text-sm">
                      YOLOモードは全てのコマンドを安全性チェックなしで実行します。
                      悪意のあるコマンドによってシステムが損傷する可能性があります。
                    </p>
                    <p className="text-sm">
                      <strong>安全な実行のために:</strong><br/>
                      Claude Codeの設定ファイルでコマンドの許可を適切に行ってください。
                    </p>
                    <p className="text-xs text-blue-600 underline">
                      <a 
                        href="https://docs.anthropic.com/ja/docs/claude-code/settings#設定ファイル" 
                        target="_blank" 
                        rel="noopener noreferrer"
                      >
                        設定方法を確認する →
                      </a>
                    </p>
                  </div>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
            <div className="text-sm text-muted-foreground">
              (確認なしで変更を実行)
            </div>
          </div>

          {/* Schedule Type */}
          <div className="space-y-2">
            <Label htmlFor="schedule-type">実行タイミング</Label>
            <Select value={scheduleType} onValueChange={(value) => {
              setScheduleType(value)
              setValidationError(null)
              // カスタム日時指定の場合、デフォルトで現在時刻を設定
              if (value === 'scheduled') {
                const now = new Date()
                setScheduledDate(format(now, 'yyyy-MM-dd'))
                setScheduledTime(format(now, 'HH:mm'))
              }
            }}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="immediate">{t('job.schedule.immediate')}</SelectItem>
                <SelectItem value="after_reset">{t('job.schedule.afterReset')}</SelectItem>
                <SelectItem value="delayed">{t('job.schedule.delayed')}</SelectItem>
                <SelectItem value="scheduled">{t('job.schedule.scheduled')}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Delayed Schedule Options */}
          {scheduleType === 'delayed' && (
            <div className="space-y-2">
              <Label>{t('job.timeUntil')}: {delayHours}{t('job.hoursAfter')}</Label>
              <Slider
                value={[delayHours]}
                onValueChange={([value]) => setDelayHours(value)}
                min={1}
                max={72}
                step={1}
                className="w-full"
              />
              <div className="text-sm text-muted-foreground">
                実行予定時刻: {format(new Date(Date.now() + delayHours * 60 * 60 * 1000), 'yyyy年MM月dd日 HH:mm', { locale: ja })}
              </div>
            </div>
          )}

          {/* Scheduled Date/Time Options */}
          {scheduleType === 'scheduled' && (
            <div className="space-y-2">
              <div className="grid grid-cols-2 gap-2">
                <div className="space-y-1">
                  <Label htmlFor="scheduled-date">実行日</Label>
                  <Input
                    id="scheduled-date"
                    type="date"
                    value={scheduledDate}
                    onChange={(e) => {
                      setScheduledDate(e.target.value)
                      setValidationError(null)
                    }}
                    min={format(new Date(), 'yyyy-MM-dd')}
                    max={format(new Date(new Date().setFullYear(new Date().getFullYear() + 1)), 'yyyy-MM-dd')}
                  />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="scheduled-time">実行時刻</Label>
                  <Input
                    id="scheduled-time"
                    type="time"
                    value={scheduledTime}
                    onChange={(e) => {
                      setScheduledTime(e.target.value)
                      setValidationError(null)
                    }}
                  />
                </div>
              </div>
              {scheduledDate && scheduledTime && (
                <div className="text-sm text-muted-foreground">
                  実行予定: {format(new Date(`${scheduledDate}T${scheduledTime}:00`), 'yyyy年MM月dd日 HH:mm', { locale: ja })}
                </div>
              )}
            </div>
          )}

          {/* Schedule Preview */}
          {scheduleType !== 'immediate' && (
            <Alert>
              <Clock className="h-4 w-4" />
              <AlertDescription>
                {scheduleType === 'after_reset' && t('job.resetDescription')}
                {scheduleType === 'delayed' && `${delayHours}${t('job.delayDescription')}`}
                {scheduleType === 'scheduled' && scheduledDate && scheduledTime && 
                  `${format(new Date(`${scheduledDate}T${scheduledTime}:00`), 'yyyy年MM月dd日 HH:mm', { locale: ja })}に実行されます。`}
              </AlertDescription>
            </Alert>
          )}

          {/* Validation Error Display */}
          {validationError && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{validationError}</AlertDescription>
            </Alert>
          )}

          {/* Error Display */}
          {createError && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{createError}</AlertDescription>
            </Alert>
          )}

          {/* Success Display */}
          {success && (
            <Alert>
              <AlertDescription className="text-green-600">{success}</AlertDescription>
            </Alert>
          )}

          {/* Submit Button */}
          <Button
            type="submit"
            disabled={!selectedProjectId || !command.trim() || createLoading || 
              (scheduleType === 'scheduled' && (!scheduledDate || !scheduledTime))}
            className="w-full"
          >
            {createLoading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
{t('job.executing')}
              </>
            ) : (
              <>
                <Play className="mr-2 h-4 w-4" />
{t('job.create')}
              </>
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}