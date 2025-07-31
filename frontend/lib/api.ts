const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:6060/api'

// Debug logging for API base URL
if (typeof window !== 'undefined') {
  console.log('CCDash API Base URL:', API_BASE_URL)
}

export interface TokenUsage {
  total_tokens: number
  input_tokens: number
  output_tokens: number
  usage_limit: number
  usage_rate: number
  window_start: string
  window_end: string
  active_sessions: number
  total_cost: number
  total_messages: number
}

export interface Session {
  id: string
  project_name: string
  project_path: string
  start_time: string
  end_time: string | null
  total_input_tokens: number
  total_output_tokens: number
  total_tokens: number
  message_count: number
  status: string
  created_at: string
  total_cost: number
  duration?: number
  is_active: boolean
  last_activity: string
  generated_code: string[]
}

export interface Message {
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
}

export interface PaginatedMessages {
  messages: Message[]
  total: number
  page: number
  page_size: number
  total_pages: number
  has_next: boolean
  has_previous: boolean
}

export interface SessionDetail {
  session: Session
  messages: Message[] | PaginatedMessages
  token_usage: TokenUsage
}

export interface P90Prediction {
  token_limit: number
  message_limit: number
  cost_limit: number
  confidence: number
  time_to_limit_minutes: number
  burn_rate_per_hour: number
  predicted_at: string
}

export interface BurnRatePoint {
  timestamp: string
  tokens_per_hour: number
}

export interface InitializationStatus {
  status: 'initializing' | 'completed' | 'failed'
  message: string
  progress?: {
    processed_files: number
    total_files: number
    new_lines: number
  }
  start_time: string
  end_time?: string
  error?: string
}

export interface ClaudeCommand {
  name: string
  description: string
  example: string
}

export interface ClaudeCommandRequest {
  session_id: string
  command: string
  timeout?: number
  yolo_mode?: boolean
}

export interface ClaudeCommandResponse {
  session_id: string
  command: string
  output: string
  error?: string
  exit_code: number
  duration_ms: number
  success: boolean
}

// Job related interfaces for async execution
export type JobStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled'

export interface Job {
  id: string
  session_id: string
  command: string
  status: JobStatus
  output: string
  error?: string
  progress: number // 0-100
  start_time: string
  end_time?: string
  duration_ms: number
  project_path?: string
  yolo_mode?: boolean
}

export interface AsyncCommandResponse {
  job_id: string
  session_id: string
  command: string
  status: JobStatus
  message: string
}

export interface ApiResponse<T> {
  data?: T
  error?: string
  message?: string
}

class ApiClient {
  private baseURL: string

  constructor(baseURL: string) {
    this.baseURL = baseURL
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseURL}${endpoint}`
    
    // Debug logging for API requests
    if (typeof window !== 'undefined') {
      console.log('CCDash API Request:', url)
    }
    
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    return response.json()
  }

  async getTokenUsage(): Promise<TokenUsage> {
    return this.request<TokenUsage>('/token-usage')
  }

  async getSessions(): Promise<{ sessions: Session[], count: number }> {
    return this.request<{ sessions: Session[], count: number }>('/claude/sessions/recent')
  }

  async getSessionDetail(sessionId: string, page?: number, pageSize?: number): Promise<SessionDetail> {
    let url = `/sessions/${sessionId}`
    if (page !== undefined || pageSize !== undefined) {
      const params = new URLSearchParams()
      if (page !== undefined) params.append('page', page.toString())
      if (pageSize !== undefined) params.append('page_size', pageSize.toString())
      url += `?${params.toString()}`
    }
    return this.request<SessionDetail>(url)
  }

  async getAvailableTokens(plan: string = 'pro'): Promise<{
    available_tokens: number
    plan: string
    usage_limit: number
    used_tokens: number
  }> {
    return this.request(`/claude/available-tokens?plan=${plan}`)
  }

  async getCurrentMonthCosts(): Promise<{
    current_month_cost: number
    currency: string
    note: string
  }> {
    return this.request('/costs/current-month')
  }

  async getTasks(): Promise<{
    tasks: unknown[]
    count: number
    note: string
  }> {
    return this.request('/tasks')
  }

  async syncLogs(): Promise<{ message: string }> {
    return this.request('/sync-logs', { method: 'POST' })
  }

  async getP90Predictions(): Promise<P90Prediction> {
    return this.request<P90Prediction>('/predictions/p90')
  }

  async getP90PredictionsByProject(projectName: string): Promise<P90Prediction> {
    return this.request<P90Prediction>(`/predictions/p90/project/${encodeURIComponent(projectName)}`)
  }

  async getBurnRateHistory(hours: number = 24): Promise<{ burn_rate_history: BurnRatePoint[], hours: number }> {
    return this.request(`/predictions/burn-rate-history?hours=${hours}`)
  }

  async getInitializationStatus(): Promise<InitializationStatus> {
    return this.request<InitializationStatus>('/initialization-status')
  }

  async getAvailableClaudeCommands(): Promise<{ commands: ClaudeCommand[], note: string }> {
    return this.request<{ commands: ClaudeCommand[], note: string }>('/claude/commands')
  }

  async executeClaudeCommand(request: ClaudeCommandRequest): Promise<ClaudeCommandResponse> {
    return this.request<ClaudeCommandResponse>('/claude/execute', {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  // Async job execution methods
  async executeClaudeCommandAsync(request: ClaudeCommandRequest): Promise<AsyncCommandResponse> {
    return this.request<AsyncCommandResponse>('/claude/execute-async', {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async getJob(jobId: string): Promise<Job> {
    return this.request<Job>(`/claude/jobs/${jobId}`)
  }

  async getAllJobs(): Promise<{ jobs: Job[], count: number }> {
    return this.request<{ jobs: Job[], count: number }>('/claude/jobs')
  }

  async getJobsBySession(sessionId: string): Promise<{ jobs: Job[], count: number, session_id: string }> {
    return this.request<{ jobs: Job[], count: number, session_id: string }>(`/claude/sessions/${sessionId}/jobs`)
  }

  async cancelJob(jobId: string): Promise<{ message: string, job_id: string }> {
    return this.request<{ message: string, job_id: string }>(`/claude/jobs/${jobId}/cancel`, {
      method: 'POST',
    })
  }

  async deleteJob(jobId: string): Promise<{ message: string, job_id: string }> {
    return this.request<{ message: string, job_id: string }>(`/claude/jobs/${jobId}`, {
      method: 'DELETE',
    })
  }

}

export const apiClient = new ApiClient(API_BASE_URL)

export const api = {
  tokenUsage: {
    getCurrent: () => apiClient.getTokenUsage(),
    getAvailable: (plan?: string) => apiClient.getAvailableTokens(plan),
  },
  sessions: {
    getAll: () => apiClient.getSessions(),
    getById: (id: string, page?: number, pageSize?: number) => apiClient.getSessionDetail(id, page, pageSize),
  },
  costs: {
    getCurrentMonth: () => apiClient.getCurrentMonthCosts(),
  },
  tasks: {
    getAll: () => apiClient.getTasks(),
  },
  sync: {
    logs: () => apiClient.syncLogs(),
  },
  predictions: {
    getP90: () => apiClient.getP90Predictions(),
    getP90ByProject: (projectName: string) => apiClient.getP90PredictionsByProject(projectName),
    getBurnRateHistory: (hours?: number) => apiClient.getBurnRateHistory(hours),
  },
  initialization: {
    getStatus: () => apiClient.getInitializationStatus(),
  },
  claude: {
    getCommands: () => apiClient.getAvailableClaudeCommands(),
    executeCommand: (request: ClaudeCommandRequest) => apiClient.executeClaudeCommand(request),
    executeCommandAsync: (request: ClaudeCommandRequest) => apiClient.executeClaudeCommandAsync(request),
    getJob: (jobId: string) => apiClient.getJob(jobId),
    getAllJobs: () => apiClient.getAllJobs(),
    getJobsBySession: (sessionId: string) => apiClient.getJobsBySession(sessionId),
    cancelJob: (jobId: string) => apiClient.cancelJob(jobId),
    deleteJob: (jobId: string) => apiClient.deleteJob(jobId),
  },
}