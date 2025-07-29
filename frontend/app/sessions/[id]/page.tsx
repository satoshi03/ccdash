"use client"

import { useState, useEffect, Suspense } from "react"
import { useParams, useRouter, useSearchParams } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination"
import { ArrowLeft, Clock, MessageSquare, Code2, User, Bot, Copy, Check } from "lucide-react"
import { api, PaginatedMessages, SessionDetail } from "@/lib/api"
import { Header } from "@/components/header"
import { useI18n } from "@/hooks/use-i18n"

type Message = {
  id: string
  session_id: string
  parent_uuid: string | null
  is_sidechain: boolean
  user_type: string | null
  message_type: string | null
  message_role: string | null
  model: string | null
  content: string | null
  input_tokens: number
  cache_creation_input_tokens: number
  cache_read_input_tokens: number
  output_tokens: number
  service_tier: string | null
  request_id: string | null
  timestamp: string
  created_at: string
};


function SessionDetailContent() {
  const { t, formatFullDate } = useI18n()
  const params = useParams()
  const router = useRouter()
  const searchParams = useSearchParams()
  const sessionId = params.id as string

  const [sessionDetail, setSessionDetail] = useState<SessionDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(20)
  const [copiedSessionId, setCopiedSessionId] = useState(false)

  useEffect(() => {
    const page = parseInt(searchParams.get('page') || '1')
    setCurrentPage(page)
  }, [searchParams])

  useEffect(() => {
    const fetchSessionDetail = async () => {
      try {
        setLoading(true)
        setError(null)
        const result = await api.sessions.getById(sessionId, currentPage, pageSize)
        setSessionDetail(result)
      } catch (err) {
        console.error('Error fetching session detail:', err)
        setError(err instanceof Error ? err.message : 'Unknown error')
      } finally {
        setLoading(false)
      }
    }

    if (sessionId) {
      fetchSessionDetail()
    }
  }, [sessionId, currentPage, pageSize])

  const formatDuration = (startTime: string, endTime: string | null) => {
    const start = new Date(startTime)
    const end = endTime ? new Date(endTime) : new Date()
    const duration = end.getTime() - start.getTime()
    const minutes = Math.floor(duration / (1000 * 60))
    const hours = Math.floor(minutes / 60)

    if (hours > 0) {
      const remainingMinutes = minutes % 60
      return `${hours} ${t('tokenUsage.hours')} ${remainingMinutes} ${t('tokenUsage.minutes')}`
    }
    return `${minutes} ${t('tokenUsage.minutes')}`
  }

  const formatTimestamp = (timestamp: string) => {
    return formatFullDate(new Date(timestamp))
  }

  const extractCodeFromContent = (content: string | null): string[] => {
    if (!content) return []
    
    const codeBlockRegex = /```[\s\S]*?```/g
    const matches = content.match(codeBlockRegex) || []
    return matches.map(match => match.replace(/```\w*\n?|\n```/g, '').trim())
  }

  const renderMessageContent = (message: Message) => {
    if (!message.content) return <div className="text-muted-foreground text-sm">{t('session.noContent')}</div>

    // Safety check for content type
    if (typeof message.content === 'object' && message.content !== null) {
      // If content is already an object, stringify it
      return (
        <div className="space-y-2">
          <div className="text-xs text-muted-foreground">{t('session.rawObjectContent')}</div>
          <pre className="bg-muted p-3 rounded-md text-xs max-w-full min-w-0">
            {JSON.stringify(message.content, null, 2)}
          </pre>
        </div>
      )
    }

    // Try to parse as JSON first (for tool use messages)
    if (typeof message.content === 'string' && message.content.trim().startsWith('[') && message.content.trim().endsWith(']')) {
      try {
        const parsed = JSON.parse(message.content)
        if (Array.isArray(parsed)) {
          return (
            <div className="space-y-2">
              {parsed.map((item: unknown, index: number) => {
                const itemObj = item as Record<string, unknown>
                return (
                  <div key={index} className="bg-muted p-3 rounded-md">
                    {itemObj.type === 'text' && (
                      <div className="whitespace-pre-wrap break-words text-sm max-w-full overflow-hidden">{itemObj.text as string}</div>
                    )}
                    {itemObj.type === 'tool_use' && (
                      <div>
                        <div className="flex items-center gap-2 mb-2">
                          <Code2 className="w-4 h-4" />
                          <span className="font-medium">{itemObj.name as string}</span>
                        </div>
                        <pre className="bg-background p-2 rounded overflow-x-scroll max-w-full min-w-0 text-xs whitespace-pre-wrap">
                          {JSON.stringify(itemObj.input, null, 2)}
                        </pre>
                      </div>
                    )}
                    {itemObj.type === 'tool_result' && (
                      <div>
                        <div className="flex items-center gap-2 mb-2">
                          <Code2 className="w-4 h-4" />
                          <span className="font-medium">{t('session.toolResult')}</span>
                          {(itemObj.tool_use_id && typeof itemObj.tool_use_id === 'string') ? (
                            <span className="text-xs text-muted-foreground">({itemObj.tool_use_id.slice(-8)})</span>
                          ) : null}
                        </div>
                        <div className="text-xs bg-background p-2 rounded whitespace-pre-wrap break-words max-w-full overflow-hidden">
                          {typeof itemObj.content === 'string' ? itemObj.content : JSON.stringify(itemObj.content, null, 2) as string}
                        </div>
                      </div>
                    )}
                    {itemObj.type === 'thinking' && (
                      <div>
                        <div className="flex items-center gap-2 mb-2">
                          <span className="font-medium text-purple-600">💭 {t('session.thinking')}</span>
                        </div>
                        <div className="text-xs bg-purple-50 p-2 rounded whitespace-pre-wrap">
                          {String(itemObj.thinking)}
                        </div>
                      </div>
                    )}
                    {!itemObj.type && (
                      <div>
                        <div className="text-xs text-muted-foreground mb-2">{t('session.unknownItemType')}</div>
                        <pre className="text-xs bg-background p-2 rounded overflow-x-auto max-w-full min-w-0">
                          {JSON.stringify(itemObj, null, 2)}
                        </pre>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )
        }
      } catch {
        // If parsing fails, fall through to plain text processing
        // Only log if it looks like it should be JSON but isn't
        console.debug('Content looks like JSON but failed to parse:', message.content.substring(0, 50) + '...')
      }
    }

    // Check if it's a simple string content (common for user messages)
    const contentStr = typeof message.content === 'string' ? message.content : JSON.stringify(message.content)
    
    // Extract code blocks
    const codeBlocks = extractCodeFromContent(contentStr)
    const textWithoutCode = contentStr.replace(/```[\s\S]*?```/g, `[${t('session.codeBlocks')}]`)

    return (
      <div className="space-y-3">
        <div className="whitespace-pre-wrap break-words text-sm max-w-full overflow-hidden">{textWithoutCode}</div>
        {codeBlocks.length > 0 && (
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium">
              <Code2 className="w-4 h-4" />
{t('session.codeGenerated')} ({codeBlocks.length}個)
            </div>
            {codeBlocks.map((code, index) => (
              <pre key={index} className="bg-muted p-3 rounded-md text-xs overflow-x-auto max-w-full min-w-0">
                {code}
              </pre>
            ))}
          </div>
        )}
      </div>
    )
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
          <p>{t('errors.sessionDetailFetch')}</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <p className="text-red-600 mb-4">{t('common.error')}: {error}</p>
          <Button onClick={() => router.push('/')}>{t('common.back')}</Button>
        </div>
      </div>
    )
  }

  if (!sessionDetail) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <p className="mb-4">{t('errors.sessionNotFound')}</p>
          <Button onClick={() => router.push('/')}>{t('common.back')}</Button>
        </div>
      </div>
    )
  }

  const { session, messages } = sessionDetail
  
  // Type guard to check if messages is paginated
  const isPaginated = (messages: unknown): messages is PaginatedMessages => {
    return Boolean(messages && typeof messages === 'object' && messages !== null && 'messages' in messages && 'total' in messages)
  }
  
  const messageList = isPaginated(messages) ? messages.messages : messages
  const totalMessages = isPaginated(messages) ? messages.total : messages.length
  const paginationInfo = isPaginated(messages) ? messages : null
  
  const handlePageChange = (page: number) => {
    const url = new URL(window.location.href)
    url.searchParams.set('page', page.toString())
    router.push(url.pathname + url.search)
  }

  return (
    <div className="min-h-screen bg-background">
      <Header />
      <div className="container mx-auto max-w-7xl p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center gap-4">
          <Button variant="outline" size="sm" onClick={() => router.push('/')}>
            <ArrowLeft className="w-4 h-4 mr-2" />
{t('common.back')}
          </Button>
          <div>
            <h1 className="text-2xl font-bold">{t('session.detail')}</h1>
            <p className="text-muted-foreground">
              {session.project_path.split("/").pop()}
            </p>
          </div>
        </div>

        {/* Session Info */}
        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">{t('session.info')}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-muted-foreground">{t('session.sessionId')}</span>
                  <div className="flex items-center gap-2 mt-1">
                    <code className="font-mono text-xs bg-muted px-2 py-1 rounded flex-1 truncate">{session.id}</code>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={async () => {
                        try {
                          if (navigator.clipboard && navigator.clipboard.writeText) {
                            await navigator.clipboard.writeText(session.id)
                          } else {
                            // Fallback for older browsers or non-HTTPS contexts
                            const textArea = document.createElement('textarea')
                            textArea.value = session.id
                            document.body.appendChild(textArea)
                            textArea.select()
                            document.execCommand('copy')
                            document.body.removeChild(textArea)
                          }
                          setCopiedSessionId(true)
                          setTimeout(() => setCopiedSessionId(false), 2000)
                        } catch (error) {
                          console.error('Failed to copy session ID:', error)
                        }
                      }}
                      className="h-6 w-6 p-0"
                    >
                      {copiedSessionId ? <Check className="w-3 h-3" /> : <Copy className="w-3 h-3" />}
                    </Button>
                  </div>
                </div>
                <div>
                  <span className="text-muted-foreground">{t('session.project')}</span>
                  <p className="font-medium break-all">{session.project_path}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">{t('session.status')}</span>
                  <div className="mt-1">
                    <Badge variant={session.is_active ? "default" : "secondary"}>
                      {session.is_active ? t('session.active') : t('session.completed')}
                    </Badge>
                  </div>
                </div>
                <div>
                  <span className="text-muted-foreground">{t('session.startTime')}</span>
                  <p className="font-medium">{formatTimestamp(session.start_time)}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">{t('session.duration')}</span>
                  <p className="font-medium">{formatDuration(session.start_time, session.end_time)}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">{t('session.messageCount')}</span>
                  <p className="font-medium">{session.message_count}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">{t('session.lastActivity')}</span>
                  <p className="font-medium">{formatTimestamp(session.last_activity)}</p>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-lg">{t('session.tokenUsage')}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-muted-foreground">{t('session.total')}</span>
                  <p className="text-2xl font-bold">{session.total_tokens.toLocaleString()}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">{t('session.cost')}</span>
                  <p className="text-2xl font-bold">${session.total_cost.toFixed(4)}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">{t('session.input')} {t('tokenUsage.tokens')}</span>
                  <p className="font-medium">{session.total_input_tokens.toLocaleString()}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">{t('session.output')} {t('tokenUsage.tokens')}</span>
                  <p className="font-medium">{session.total_output_tokens.toLocaleString()}</p>
                </div>
                <div>
                  <span className="text-muted-foreground">{t('session.messages')}</span>
                  <p className="font-medium">{session.message_count}</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Messages */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <MessageSquare className="w-5 h-5" />
{t('session.messageHistory')} ({totalMessages})
            </CardTitle>
            <CardDescription>
{t('session.messageHistory')}
              {paginationInfo && (
                <span className="ml-2">
                  ({t('session.page')} {paginationInfo.page} / {paginationInfo.total_pages})
                </span>
              )}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="pr-4">
              <div className="space-y-4">
                {messageList.map((message, index) => (
                  <div key={message.id} className="border rounded-lg p-4 min-w-0 overfl">
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-2">
                        {message.message_role === 'user' ? (
                          <User className="w-4 h-4 text-blue-600" />
                        ) : (
                          <Bot className="w-4 h-4 text-green-600" />
                        )}
                        <span className="font-medium capitalize">
                          {message.message_role === 'user' ? t('session.user') : t('session.assistant')}
                        </span>
                        {message.model && (
                          <Badge variant="outline" className="text-xs">
                            {message.model}
                          </Badge>
                        )}
                      </div>
                      <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        <Clock className="w-3 h-3" />
                        {formatTimestamp(message.timestamp)}
                      </div>
                    </div>

                    {message.content && (
                      <div className="mb-3">
                        {renderMessageContent(message)}
                      </div>
                    )}

                    {message.message_role === 'assistant' && (message.input_tokens > 0 || message.output_tokens > 0) && (
                      <div className="flex items-center gap-4 text-xs text-muted-foreground border-t pt-2">
                        <span>{t('session.input')}: {message.input_tokens.toLocaleString()}</span>
                        <span>{t('session.output')}: {message.output_tokens.toLocaleString()}</span>
                        <span>{t('session.total')}: {(message.input_tokens + message.output_tokens).toLocaleString()}</span>
                        {message.service_tier && (
                          <Badge variant="outline" className="text-xs">
                            {message.service_tier}
                          </Badge>
                        )}
                      </div>
                    )}

                    {index < messageList.length - 1 && <Separator className="mt-4" />}
                  </div>
                ))}
              </div>
            </div>
            
            {/* Pagination */}
            {paginationInfo && paginationInfo.total_pages > 1 && (
              <div className="mt-6">
                <Pagination>
                  <PaginationContent>
                    <PaginationItem>
                      <PaginationPrevious
                        href="#"
                        onClick={(e) => {
                          e.preventDefault()
                          if (paginationInfo.has_previous) {
                            handlePageChange(paginationInfo.page - 1)
                          }
                        }}
                        className={!paginationInfo.has_previous ? 'pointer-events-none opacity-50' : ''}
                      />
                    </PaginationItem>
                    
                    {/* Page numbers */}
                    {(() => {
                      const pages = []
                      const current = paginationInfo.page
                      const total = paginationInfo.total_pages
                      
                      // Always show first page
                      if (current > 3) {
                        pages.push(
                          <PaginationItem key={1}>
                            <PaginationLink
                              href="#"
                              onClick={(e) => {
                                e.preventDefault()
                                handlePageChange(1)
                              }}
                            >
                              1
                            </PaginationLink>
                          </PaginationItem>
                        )
                        if (current > 4) {
                          pages.push(<PaginationItem key="ellipsis1"><PaginationEllipsis /></PaginationItem>)
                        }
                      }
                      
                      // Show pages around current
                      const start = Math.max(1, current - 2)
                      const end = Math.min(total, current + 2)
                      
                      for (let i = start; i <= end; i++) {
                        pages.push(
                          <PaginationItem key={i}>
                            <PaginationLink
                              href="#"
                              isActive={i === current}
                              onClick={(e) => {
                                e.preventDefault()
                                handlePageChange(i)
                              }}
                            >
                              {i}
                            </PaginationLink>
                          </PaginationItem>
                        )
                      }
                      
                      // Always show last page
                      if (current < total - 2) {
                        if (current < total - 3) {
                          pages.push(<PaginationItem key="ellipsis2"><PaginationEllipsis /></PaginationItem>)
                        }
                        pages.push(
                          <PaginationItem key={total}>
                            <PaginationLink
                              href="#"
                              onClick={(e) => {
                                e.preventDefault()
                                handlePageChange(total)
                              }}
                            >
                              {total}
                            </PaginationLink>
                          </PaginationItem>
                        )
                      }
                      
                      return pages
                    })()}
                    
                    <PaginationItem>
                      <PaginationNext
                        href="#"
                        onClick={(e) => {
                          e.preventDefault()
                          if (paginationInfo.has_next) {
                            handlePageChange(paginationInfo.page + 1)
                          }
                        }}
                        className={!paginationInfo.has_next ? 'pointer-events-none opacity-50' : ''}
                      />
                    </PaginationItem>
                  </PaginationContent>
                </Pagination>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function LoadingFallback() {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className="text-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
        <p>Loading session details...</p>
      </div>
    </div>
  )
}

export default function SessionDetailPage() {
  return (
    <Suspense fallback={<LoadingFallback />}>
      <SessionDetailContent />
    </Suspense>
  )
}