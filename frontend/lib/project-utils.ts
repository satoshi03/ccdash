import { Session } from '@/lib/api'

export type ConvertedProject = {
  id: string
  name: string
  originalPath: string
  sessions: Array<{
    id: string
    sessionId: string
    startTime: Date
    endTime: Date | null
    tokenUsage: number
    status: 'running' | 'completed' | 'paused' | 'failed'
    messageCount: number
    codeGenerated: boolean
  }>
}

export function convertSessionsToProjects(sessions: Session[]): ConvertedProject[] {
  const projectMap = new Map<string, ConvertedProject>()
  
  sessions.forEach(session => {
    const projectPath = session.project_path
    const projectName = session.project_name
    
    if (!projectMap.has(projectPath)) {
      projectMap.set(projectPath, {
        id: projectPath, // プロジェクトパスを一意のIDとして使用
        name: projectName,
        originalPath: projectPath,
        sessions: []
      })
    }
    
    const project = projectMap.get(projectPath)!
    project.sessions.push({
      id: `${session.id}-${session.start_time}`, // より一意性を高める
      sessionId: session.id,
      startTime: new Date(session.start_time),
      endTime: session.end_time ? new Date(session.end_time) : null,
      tokenUsage: session.total_tokens,
      status: session.is_active ? 'running' : 'completed',
      messageCount: session.message_count,
      codeGenerated: session.generated_code?.length > 0
    })
  })
  
  // プロジェクトを最終実行時間でソート
  const projectsArray = Array.from(projectMap.values())
  projectsArray.forEach(project => {
    // 各プロジェクト内のセッションを最終実行時間でソート
    project.sessions.sort((a, b) => {
      const aTime = a.endTime || a.startTime
      const bTime = b.endTime || b.startTime
      return bTime.getTime() - aTime.getTime()
    })
  })
  
  // プロジェクト自体も最終実行時間でソート
  projectsArray.sort((a, b) => {
    const aLastTime = a.sessions.length > 0 ? (a.sessions[0].endTime || a.sessions[0].startTime) : new Date(0)
    const bLastTime = b.sessions.length > 0 ? (b.sessions[0].endTime || b.sessions[0].startTime) : new Date(0)
    return bLastTime.getTime() - aLastTime.getTime()
  })
  
  return projectsArray
}