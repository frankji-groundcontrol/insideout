import { getSupabase } from '@/lib/supabase'
import {
  AiRateLimitError,
  AiServiceUnavailableError,
  type AiConversation,
  type AiMessage,
} from '@/types'
import type { IAiService } from '@/types/services'

const CONVERSATION_PREFIX = 'ai-conv:'

type GenerateResponse = {
  id: string
  role: 'assistant'
  content: string
  timestamp: string
}

type GenerateErrorResponse = {
  error?: string
  code?: string
  retry_after_seconds?: number
  current_count?: number
  max_requests?: number
  circuit_state?: string
}

function asRecord(value: unknown): Record<string, unknown> {
  return typeof value === 'object' && value !== null ? (value as Record<string, unknown>) : {}
}

function extractStatus(error: unknown): number | undefined {
  const raw = asRecord(error)
  const context = asRecord(raw.context)
  const status = context.status
  return typeof status === 'number' ? status : undefined
}

function parseGenerateError(data: unknown): GenerateErrorResponse {
  const raw = asRecord(data)
  return {
    error: typeof raw.error === 'string' ? raw.error : undefined,
    code: typeof raw.code === 'string' ? raw.code : undefined,
    retry_after_seconds:
      typeof raw.retry_after_seconds === 'number' ? raw.retry_after_seconds : undefined,
    current_count: typeof raw.current_count === 'number' ? raw.current_count : undefined,
    max_requests: typeof raw.max_requests === 'number' ? raw.max_requests : undefined,
    circuit_state: typeof raw.circuit_state === 'string' ? raw.circuit_state : undefined,
  }
}

function buildUserMessage(message: string): AiMessage {
  return {
    id: `msg_${Date.now()}_user`,
    role: 'user',
    content: message,
    timestamp: new Date().toISOString(),
  }
}

function validateAssistantMessage(data: unknown): AiMessage {
  const payload = data as Partial<GenerateResponse>
  if (
    !payload ||
    typeof payload.id !== 'string' ||
    payload.role !== 'assistant' ||
    typeof payload.content !== 'string' ||
    typeof payload.timestamp !== 'string'
  ) {
    throw new Error('Edge Function 返回格式不正确')
  }

  return {
    id: payload.id,
    role: 'assistant',
    content: payload.content,
    timestamp: payload.timestamp,
  }
}

export function createSupabaseAiService(): IAiService {
  return {
    async reply(nodeId: string, message: string): Promise<AiMessage> {
      const supabase = getSupabase()

      // 先调用服务端函数生成回复，再把用户消息与助手消息一起落到本地会话。
      const { data, error } = await supabase.functions.invoke('juanleme-ai-generate', {
        body: { nodeId, message },
      })

      if (error) {
        const status = extractStatus(error)
        const payload = parseGenerateError(data)
        if (status === 429) {
          throw new AiRateLimitError(
            payload.error ?? 'Rate limit exceeded',
            payload.retry_after_seconds ?? 60,
            payload.current_count,
            payload.max_requests,
          )
        }

        if (status === 503) {
          throw new AiServiceUnavailableError(
            payload.error ?? 'AI service temporarily unavailable',
            payload.retry_after_seconds ?? 30,
            payload.circuit_state,
          )
        }

        throw new Error(error.message)
      }

      const assistantMessage = validateAssistantMessage(data)
      const userMessage = buildUserMessage(message)
      const conv = await this.getConversation(nodeId)
      const messages = conv
        ? [...conv.messages, userMessage, assistantMessage]
        : [userMessage, assistantMessage]

      localStorage.setItem(
        `${CONVERSATION_PREFIX}${nodeId}`,
        JSON.stringify({ nodeId, messages }),
      )

      return assistantMessage
    },

    async getConversation(nodeId: string): Promise<AiConversation | undefined> {
      const raw = localStorage.getItem(`${CONVERSATION_PREFIX}${nodeId}`)
      if (!raw) {
        return undefined
      }

      try {
        return JSON.parse(raw) as AiConversation
      } catch {
        // 本地缓存损坏时清理，避免后续反复解析失败。
        localStorage.removeItem(`${CONVERSATION_PREFIX}${nodeId}`)
        return undefined
      }
    },

    async clearConversation(nodeId: string): Promise<void> {
      localStorage.removeItem(`${CONVERSATION_PREFIX}${nodeId}`)
    },
  }
}
