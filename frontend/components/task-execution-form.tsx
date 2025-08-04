'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Loader2, Play, AlertCircle, Clock } from 'lucide-react'
import { useProjects, useCreateJob } from '@/hooks/use-job-api'
import { CreateJobRequest } from '@/lib/api'
import { Slider } from '@/components/ui/slider'
import { Input } from '@/components/ui/input'
import { format } from 'date-fns'
import { ja } from 'date-fns/locale'

interface TaskExecutionFormProps {
  onJobCreated?: () => void
}

export function TaskExecutionForm({ onJobCreated }: TaskExecutionFormProps) {
  const { projects, loading: projectsLoading } = useProjects()
  const { createJob, loading: createLoading, error: createError } = useCreateJob()
  
  const [selectedProjectId, setSelectedProjectId] = useState<string>('')
  const [command, setCommand] = useState('')
  const [yoloMode, setYoloMode] = useState(false)
  const [scheduleType, setScheduleType] = useState('immediate')
  const [delayHours, setDelayHours] = useState(1)
  const [scheduledDate, setScheduledDate] = useState('')
  const [scheduledTime, setScheduledTime] = useState('')
  const [success, setSuccess] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!selectedProjectId || !command.trim()) {
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
      setSuccess(`Job created successfully: ${job.id}`)
      
      // Reset form
      setCommand('')
      setYoloMode(false)
      setScheduleType('immediate')
      setDelayHours(1)
      setScheduledDate('')
      setScheduledTime('')
      
      // Notify parent component
      if (onJobCreated) {
        onJobCreated()
      }
    } catch {
      // Error is handled by the hook
    }
  }

  const selectedProject = projects.find(p => p.id === selectedProjectId)

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Play className="h-5 w-5" />
          タスク実行
        </CardTitle>
        <CardDescription>
          Claude Codeタスクを実行します。プロジェクトを選択してコマンドを入力してください。
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Project Selection */}
          <div className="space-y-2">
            <Label htmlFor="project">プロジェクト</Label>
            <Select value={selectedProjectId} onValueChange={setSelectedProjectId}>
              <SelectTrigger>
                <SelectValue placeholder="プロジェクトを選択してください" />
              </SelectTrigger>
              <SelectContent>
                {projectsLoading ? (
                  <SelectItem value="loading" disabled>
                    読み込み中...
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
            <Label htmlFor="command">コマンド</Label>
            <Textarea
              id="command"
              placeholder="例: implement a new feature to..."
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
            <div className="text-sm text-muted-foreground">
              (確認なしで変更を実行)
            </div>
          </div>

          {/* Schedule Type */}
          <div className="space-y-2">
            <Label htmlFor="schedule-type">実行タイミング</Label>
            <Select value={scheduleType} onValueChange={setScheduleType}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="immediate">即座に実行</SelectItem>
                <SelectItem value="after_reset">次回リセット後</SelectItem>
                <SelectItem value="delayed">N時間後に実行</SelectItem>
                <SelectItem value="scheduled">日時を指定</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Delayed Schedule Options */}
          {scheduleType === 'delayed' && (
            <div className="space-y-2">
              <Label>実行まで: {delayHours}時間後</Label>
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
                    onChange={(e) => setScheduledDate(e.target.value)}
                    min={format(new Date(), 'yyyy-MM-dd')}
                  />
                </div>
                <div className="space-y-1">
                  <Label htmlFor="scheduled-time">実行時刻</Label>
                  <Input
                    id="scheduled-time"
                    type="time"
                    value={scheduledTime}
                    onChange={(e) => setScheduledTime(e.target.value)}
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
                {scheduleType === 'after_reset' && 'セッションウィンドウがリセットされた後に実行されます。'}
                {scheduleType === 'delayed' && `${delayHours}時間後に実行されます。`}
                {scheduleType === 'scheduled' && scheduledDate && scheduledTime && 
                  `${format(new Date(`${scheduledDate}T${scheduledTime}:00`), 'yyyy年MM月dd日 HH:mm', { locale: ja })}に実行されます。`}
              </AlertDescription>
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
                実行中...
              </>
            ) : (
              <>
                <Play className="mr-2 h-4 w-4" />
                タスクを実行
              </>
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}