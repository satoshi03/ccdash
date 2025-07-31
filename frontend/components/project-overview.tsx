import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Folder } from "lucide-react"
import Link from "next/link"
import { useI18n } from "@/hooks/use-i18n"

interface Session {
  id: string
  sessionId: string
  startTime: Date
  endTime: Date | null
  tokenUsage: number
  status: "completed" | "running" | "paused" | "failed"
  messageCount: number
  codeGenerated: boolean
}

interface Project {
  id: string
  name: string
  originalPath: string
  sessions: Session[]
}

interface ProjectOverviewProps {
  projects: Project[]
}

export function ProjectOverview({ projects }: ProjectOverviewProps) {
  const { t, formatDate } = useI18n()
  
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

  if (projects.length === 0) {
    return (
      <div className="text-center py-8">
        <Folder className="w-12 h-12 mx-auto text-muted-foreground mb-4" />
        <p className="text-lg font-medium text-muted-foreground">{t('empty.noProjects')}</p>
        <p className="text-sm text-muted-foreground">{t('empty.noProjectsDesc')}</p>
      </div>
    )
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {projects.map((project) => {
        const totalTokens = project.sessions.reduce((sum, session) => sum + session.tokenUsage, 0)
        const activeSessions = project.sessions.filter((s) => s.status === "running").length
        const completedSessions = project.sessions.filter((s) => s.status === "completed").length

        return (
          <Card key={project.id} className="hover:shadow-md transition-shadow">
            <CardHeader className="pb-3">
              <div className="flex items-center gap-2">
                <Folder className="w-4 h-4 text-muted-foreground" />
                <Link 
                  href={`/projects/${encodeURIComponent(project.name)}`}
                  className="hover:underline"
                >
                  <CardTitle className="text-sm font-medium truncate">{project.originalPath.split("/").pop()}</CardTitle>
                </Link>
              </div>
              <CardDescription className="text-xs">{project.originalPath}</CardDescription>
            </CardHeader>

            <CardContent className="space-y-3">
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">{t('project.totalTokenUsage')}</span>
                <span className="font-semibold">{totalTokens.toLocaleString()}</span>
              </div>

              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">{t('project.sessionCount')}</span>
                <div className="flex gap-1">
                  {activeSessions > 0 && (
                    <Badge variant="secondary" className="text-xs">
                      {t('project.running')} {activeSessions}
                    </Badge>
                  )}
                  <Badge variant="outline" className="text-xs">
                    {t('project.completed')} {completedSessions}
                  </Badge>
                </div>
              </div>

              <div className="space-y-2">
                <span className="text-sm text-muted-foreground">{t('project.recentSessions')}</span>
                {project.sessions.slice(0, 2).map((session) => (
                  <Link 
                    key={session.id} 
                    href={`/sessions/${session.sessionId}`}
                    className="block"
                  >
                    <div className="flex items-center justify-between p-2 bg-muted/50 rounded-sm hover:bg-muted transition-colors cursor-pointer group">
                      <div className="flex items-center gap-2">
                        <Badge variant="secondary" className={`text-xs ${getStatusColor(session.status)}`}>
                          {getStatusText(session.status)}
                        </Badge>
                        <span className="text-xs text-muted-foreground">
                          {session.tokenUsage.toLocaleString()} {t('project.tokensLabel')}
                        </span>
                        <span className="text-xs text-muted-foreground">
                          {session.messageCount} msgs
                        </span>
                      </div>
                      <div className="flex items-center gap-1 text-xs text-muted-foreground">
                        {session.endTime
                          ? formatDate(new Date(session.endTime))
                          : t('session.active')}
                      </div>
                    </div>
                  </Link>
                ))}
                {project.sessions.length > 2 && (
                  <div className="text-xs text-muted-foreground text-center py-1">
                    {t('project.others')} {project.sessions.length - 2} {t('project.sessionsText')}
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
