export interface UserProfile {
  id: string
  email: string
  username: string
  avatar_url?: string
  bio?: string
  keywords?: string[]
  role: 'admin' | 'user'
  created_at: string
}

export interface Workshop {
  id: string
  title: string
  description: string
  cover_url?: string
  code: string // 6位邀请码
  creator_id: string
  status: 'draft' | 'active' | 'completed'
  created_at: string
  member_count: number
  is_joined?: boolean // 当前用户是否已加入
}

export interface RoadmapNode {
  id: string
  workshop_id: string
  title: string
  description: string
  status: 'locked' | 'pending' | 'in_progress' | 'completed'
  order: number
  content?: string // 用户填写的内容
}

// 编辑器草稿
export interface EditorDraft {
  key: string
  content: unknown
  updatedAt: string
  revision: number
}

// AI 消息
export interface AiMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  timestamp: string
}

// AI 对话
export interface AiConversation {
  nodeId: string
  messages: AiMessage[]
}

export class AiRateLimitError extends Error {
  retryAfterSeconds: number
  currentCount?: number
  maxRequests?: number

  constructor(
    message: string,
    retryAfterSeconds: number,
    currentCount?: number,
    maxRequests?: number,
  ) {
    super(message)
    this.name = 'AiRateLimitError'
    this.retryAfterSeconds = retryAfterSeconds
    this.currentCount = currentCount
    this.maxRequests = maxRequests
  }
}

export class AiServiceUnavailableError extends Error {
  retryAfterSeconds: number
  circuitState?: string

  constructor(message: string, retryAfterSeconds: number, circuitState?: string) {
    super(message)
    this.name = 'AiServiceUnavailableError'
    this.retryAfterSeconds = retryAfterSeconds
    this.circuitState = circuitState
  }
}

// 导出配置
export interface ExportConfig {
  workshopId: string
  format: 'markdown' | 'print'
  includeEmptyNodes: boolean
}

// 用户提交内容
export interface UserSubmission {
  nodeId: string
  userId: string
  content: unknown
  submittedAt: string
}
