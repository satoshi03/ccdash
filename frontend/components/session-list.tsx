"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import { ChevronDown, ChevronRight, Play, Pause, Square, MessageSquare } from "lucide-react"
import { useI18n } from "@/hooks/use-i18n"

interface Session {
  id: string
  sessionId: string
  startTime: Date
  endTime: Date | null
  tokenUsage: number
  status: "completed" | "running" | "paused" | "failed"
  messageCount: number
}

interface Project {
  id: string
  name: string
  originalPath: string
  sessions: Session[]
}

interface SessionListProps {
  projects: Project[]
}

export function SessionList({ projects }: SessionListProps) {
  const [expandedProjects, setExpandedProjects] = useState<Set<string>>(new Set())
  const router = useRouter()
  const { t, formatDate } = useI18n()

  const toggleProject = (projectId: string) => {
    const newExpanded = new Set(expandedProjects)
    if (newExpanded.has(projectId)) {
      newExpanded.delete(projectId)
    } else {
      newExpanded.add(projectId)
    }
    setExpandedProjects(newExpanded)
  }

  const getStatusIcon = (status: Session["status"]) => {
    switch (status) {
      case "completed":
        return <Square className="w-4 h-4 text-green-600" />
      case "running":
        return <Play className="w-4 h-4 text-blue-600" />
      case "paused":
        return <Pause className="w-4 h-4 text-yellow-600" />
      case "failed":
        return <Square className="w-4 h-4 text-red-600" />
      default:
        return <Square className="w-4 h-4 text-gray-600" />
    }
  }

  const getStatusText = (status: Session["status"]) => {
    switch (status) {
      case "completed":
        return t('session.completed')
      case "running":
        return t('session.active')
      case "paused":
        return t('session.paused')
      case "failed":
        return t('session.failed')
      default:
        return t('session.unknown')
    }
  }

  const getStatusColor = (status: Session["status"]) => {
    switch (status) {
      case "completed":
        return "bg-green-100 text-green-800"
      case "running":
        return "bg-blue-100 text-blue-800"
      case "paused":
        return "bg-yellow-100 text-yellow-800"
      case "failed":
        return "bg-red-100 text-red-800"
      default:
        return "bg-gray-100 text-gray-800"
    }
  }

  const formatDuration = (startTime: Date, endTime: Date | null) => {
    const end = endTime || new Date()
    const duration = end.getTime() - startTime.getTime()
    const minutes = Math.floor(duration / (1000 * 60))
    const hours = Math.floor(minutes / 60)

    if (hours > 0) {
      return `${hours}${t('tokenUsage.hours')}${minutes % 60}${t('tokenUsage.minutes')}`
    }
    return `${minutes}${t('tokenUsage.minutes')}`
  }

  if (projects.length === 0) {
    return (
      <div className="text-center py-8">
        <div className="w-12 h-12 mx-auto text-muted-foreground mb-4">
          <MessageSquare className="w-full h-full" />
        </div>
        <p className="text-lg font-medium text-muted-foreground">{t('empty.noSessions')}</p>
        <p className="text-sm text-muted-foreground">{t('empty.noSessionsDesc')}</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {projects.map((project) => (
        <Card key={project.id}>
          <Collapsible open={expandedProjects.has(project.id)} onOpenChange={() => toggleProject(project.id)}>
            <CollapsibleTrigger asChild>
              <CardHeader className="cursor-pointer hover:bg-muted/50 transition-colors">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    {expandedProjects.has(project.id) ? (
                      <ChevronDown className="w-4 h-4" />
                    ) : (
                      <ChevronRight className="w-4 h-4" />
                    )}
                    <div>
                      <CardTitle className="text-base">{project.originalPath.split("/").pop()}</CardTitle>
                      <CardDescription className="text-sm">{project.originalPath}</CardDescription>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline">{project.sessions.length} {t('common.sessions')}</Badge>
                    <Badge variant="secondary">
                      {project.sessions.reduce((sum, s) => sum + s.tokenUsage, 0).toLocaleString()} tokens
                    </Badge>
                  </div>
                </div>
              </CardHeader>
            </CollapsibleTrigger>

            <CollapsibleContent>
              <CardContent className="pt-0">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('session.status')}</TableHead>
                      <TableHead>{t('session.sessionId')}</TableHead>
                      <TableHead>{t('session.startTime')}</TableHead>
                      <TableHead>{t('session.duration')}</TableHead>
                      <TableHead>{t('session.tokenUsage')}</TableHead>
                      <TableHead>{t('session.messageCount')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {project.sessions.map((session) => (
                      <TableRow 
                        key={session.id}
                        className="cursor-pointer hover:bg-muted/50 transition-colors"
                        onClick={() => router.push(`/sessions/${session.sessionId}`)}
                      >
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {getStatusIcon(session.status)}
                            <Badge variant="secondary" className={`text-xs ${getStatusColor(session.status)}`}>
                              {getStatusText(session.status)}
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell className="font-mono text-sm">{session.sessionId}</TableCell>
                        <TableCell>
                          {formatDate(session.startTime)}
                        </TableCell>
                        <TableCell>{formatDuration(session.startTime, session.endTime)}</TableCell>
                        <TableCell className="font-semibold">{session.tokenUsage.toLocaleString()}</TableCell>
                        <TableCell>
                          {session.messageCount}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </CardContent>
            </CollapsibleContent>
          </Collapsible>
        </Card>
      ))}
    </div>
  )
}
