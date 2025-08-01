'use client'

import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Loader2, Play, AlertCircle } from 'lucide-react'
import { useProjects, useCreateJob } from '@/hooks/use-job-api'
import { CreateJobRequest } from '@/lib/api'

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

      const job = await createJob(request)
      setSuccess(`Job created successfully: ${job.id}`)
      
      // Reset form
      setCommand('')
      setYoloMode(false)
      setScheduleType('immediate')
      
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
                <SelectItem value="custom">カスタム</SelectItem>
              </SelectContent>
            </Select>
          </div>

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
            disabled={!selectedProjectId || !command.trim() || createLoading}
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