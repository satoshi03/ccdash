"use client"

import { useState } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Plus, Play, Trash2, Clock, Calendar, AlertCircle } from "lucide-react"
import { useI18n } from "@/hooks/use-i18n"

interface Task {
  id: string
  name: string
  description: string
  projectPath: string
  estimatedTokens: number
  priority: "high" | "medium" | "low"
  status: "waiting" | "running" | "completed" | "failed"
  createdAt: Date
  scheduledAt: Date | null
}

interface TaskSchedulerProps {
  resetTime: Date
}

export function TaskScheduler({ resetTime }: TaskSchedulerProps) {
  const { formatFullDate } = useI18n()
  const [tasks, setTasks] = useState<Task[]>([
    {
      id: "1",
      name: "APIドキュメント生成",
      description: "REST APIの詳細なドキュメントを生成する",
      projectPath: "/Users/satoshi/git/api-project",
      estimatedTokens: 2500,
      priority: "high",
      status: "waiting",
      createdAt: new Date(Date.now() - 2 * 60 * 60 * 1000),
      scheduledAt: resetTime,
    },
    {
      id: "2",
      name: "テストケース作成",
      description: "ユニットテストとE2Eテストのケースを作成",
      projectPath: "/Users/satoshi/projects/web-app",
      estimatedTokens: 1800,
      priority: "medium",
      status: "waiting",
      createdAt: new Date(Date.now() - 1 * 60 * 60 * 1000),
      scheduledAt: resetTime,
    },
  ])

  const [isDialogOpen, setIsDialogOpen] = useState(false)
  const [newTask, setNewTask] = useState({
    name: "",
    description: "",
    projectPath: "",
    estimatedTokens: 1000,
    priority: "medium" as const,
  })

  const addTask = () => {
    const task: Task = {
      id: Date.now().toString(),
      ...newTask,
      status: "waiting",
      createdAt: new Date(),
      scheduledAt: resetTime,
    }
    setTasks([...tasks, task])
    setNewTask({
      name: "",
      description: "",
      projectPath: "",
      estimatedTokens: 1000,
      priority: "medium",
    })
    setIsDialogOpen(false)
  }

  const deleteTask = (taskId: string) => {
    setTasks(tasks.filter((t) => t.id !== taskId))
  }

  const runTaskNow = (taskId: string) => {
    setTasks(tasks.map((t) => (t.id === taskId ? { ...t, status: "running" as const } : t)))
  }

  const getPriorityColor = (priority: Task["priority"]) => {
    switch (priority) {
      case "high":
        return "bg-red-100 text-red-800"
      case "medium":
        return "bg-yellow-100 text-yellow-800"
      case "low":
        return "bg-green-100 text-green-800"
      default:
        return "bg-gray-100 text-gray-800"
    }
  }

  const getPriorityText = (priority: Task["priority"]) => {
    switch (priority) {
      case "high":
        return "高"
      case "medium":
        return "中"
      case "low":
        return "低"
      default:
        return "不明"
    }
  }

  const getStatusColor = (status: Task["status"]) => {
    switch (status) {
      case "waiting":
        return "bg-blue-100 text-blue-800"
      case "running":
        return "bg-green-100 text-green-800"
      case "completed":
        return "bg-gray-100 text-gray-800"
      case "failed":
        return "bg-red-100 text-red-800"
      default:
        return "bg-gray-100 text-gray-800"
    }
  }

  const getStatusText = (status: Task["status"]) => {
    switch (status) {
      case "waiting":
        return "待機中"
      case "running":
        return "実行中"
      case "completed":
        return "完了"
      case "failed":
        return "失敗"
      default:
        return "不明"
    }
  }

  const totalEstimatedTokens = tasks
    .filter((t) => t.status === "waiting")
    .reduce((sum, t) => sum + t.estimatedTokens, 0)

  const timeUntilReset = Math.max(0, resetTime.getTime() - Date.now())
  const hoursUntilReset = Math.floor(timeUntilReset / (1000 * 60 * 60))
  const minutesUntilReset = Math.floor((timeUntilReset % (1000 * 60 * 60)) / (1000 * 60))

  return (
    <div className="space-y-6">
      {/* スケジューラー概要 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Calendar className="w-5 h-5" />
            タスクスケジューラー
          </CardTitle>
          <CardDescription>トークン制限リセット後に自動実行されるタスクを管理します</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="flex items-center gap-2">
              <Clock className="w-4 h-4 text-muted-foreground" />
              <div>
                <div className="text-sm text-muted-foreground">次回実行まで</div>
                <div className="font-semibold">
                  {hoursUntilReset}時間{minutesUntilReset}分
                </div>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <AlertCircle className="w-4 h-4 text-muted-foreground" />
              <div>
                <div className="text-sm text-muted-foreground">待機中タスク</div>
                <div className="font-semibold">{tasks.filter((t) => t.status === "waiting").length} 件</div>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-blue-500 rounded-full" />
              <div>
                <div className="text-sm text-muted-foreground">予想トークン使用量</div>
                <div className="font-semibold">{totalEstimatedTokens.toLocaleString()} tokens</div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* タスク一覧 */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between space-y-0">
          <div>
            <CardTitle>タスク一覧</CardTitle>
            <CardDescription>スケジュールされたタスクの管理と実行</CardDescription>
          </div>

          <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="w-4 h-4 mr-2" />
                新規タスク
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[425px]">
              <DialogHeader>
                <DialogTitle>新規タスク作成</DialogTitle>
                <DialogDescription>新しいタスクを作成してスケジュールに追加します</DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="grid gap-2">
                  <Label htmlFor="name">タスク名</Label>
                  <Input
                    id="name"
                    value={newTask.name}
                    onChange={(e) => setNewTask({ ...newTask, name: e.target.value })}
                    placeholder="例: APIドキュメント生成"
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="description">説明</Label>
                  <Textarea
                    id="description"
                    value={newTask.description}
                    onChange={(e) => setNewTask({ ...newTask, description: e.target.value })}
                    placeholder="タスクの詳細な説明を入力してください"
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="projectPath">プロジェクトパス</Label>
                  <Input
                    id="projectPath"
                    value={newTask.projectPath}
                    onChange={(e) => setNewTask({ ...newTask, projectPath: e.target.value })}
                    placeholder="/Users/username/project"
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="estimatedTokens">予想トークン数</Label>
                  <Input
                    id="estimatedTokens"
                    type="number"
                    value={newTask.estimatedTokens}
                    onChange={(e) => setNewTask({ ...newTask, estimatedTokens: Number.parseInt(e.target.value) || 0 })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="priority">優先度</Label>
                  <Select
                    value={newTask.priority}
                    onValueChange={(value: "high" | "medium" | "low") => setNewTask({ ...newTask, priority: value })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="high">高</SelectItem>
                      <SelectItem value="medium">中</SelectItem>
                      <SelectItem value="low">低</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <DialogFooter>
                <Button onClick={addTask} disabled={!newTask.name.trim()}>
                  タスクを作成
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </CardHeader>

        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>タスク名</TableHead>
                <TableHead>プロジェクト</TableHead>
                <TableHead>優先度</TableHead>
                <TableHead>ステータス</TableHead>
                <TableHead>予想トークン</TableHead>
                <TableHead>実行予定</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tasks.map((task) => (
                <TableRow key={task.id}>
                  <TableCell>
                    <div>
                      <div className="font-medium">{task.name}</div>
                      <div className="text-sm text-muted-foreground">{task.description}</div>
                    </div>
                  </TableCell>
                  <TableCell className="text-sm">{task.projectPath.split("/").pop()}</TableCell>
                  <TableCell>
                    <Badge variant="secondary" className={`text-xs ${getPriorityColor(task.priority)}`}>
                      {getPriorityText(task.priority)}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant="secondary" className={`text-xs ${getStatusColor(task.status)}`}>
                      {getStatusText(task.status)}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono">{task.estimatedTokens.toLocaleString()}</TableCell>
                  <TableCell className="text-sm">
                    {task.scheduledAt
                      ? formatFullDate(task.scheduledAt)
                      : "未設定"}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      {task.status === "waiting" && (
                        <Button size="sm" variant="outline" onClick={() => runTaskNow(task.id)}>
                          <Play className="w-3 h-3" />
                        </Button>
                      )}
                      <Button size="sm" variant="outline" onClick={() => deleteTask(task.id)}>
                        <Trash2 className="w-3 h-3" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
