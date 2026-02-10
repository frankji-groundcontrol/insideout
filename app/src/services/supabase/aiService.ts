import { getSupabase } from '@/lib/supabase'
import type { AiConversation, AiMessage } from '@/types'
import type { IAiService } from '@/types/services'

const CONVERSATION_PREFIX = 'ai-conv:'

type GenerateResponse = {
  id: string
  role: 'assistant'
  content: string
  timestamp: string
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
