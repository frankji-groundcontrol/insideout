import type { IAiService } from '@/types/services'
import type { AiMessage, AiConversation } from '@/types'
import { AI_TEMPLATES, AI_DEFAULT_RESPONSE } from './data'

const CONVERSATION_PREFIX = 'ai-conv:'

function matchTemplate(message: string): string {
  const lowerMsg = message.toLowerCase()
  const matched = AI_TEMPLATES.find(t =>
    t.keywords.some(kw => lowerMsg.includes(kw.toLowerCase())),
  )
  return matched ? matched.response : AI_DEFAULT_RESPONSE
}

function randomDelay(): Promise<void> {
  const ms = 500 + Math.random() * 1000
  return new Promise(resolve => setTimeout(resolve, ms))
}

export function createMockAiService(): IAiService {
  return {
    async reply(nodeId: string, message: string): Promise<AiMessage> {
      await randomDelay()

      const responseContent = matchTemplate(message)
      const aiMessage: AiMessage = {
        id: `msg_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`,
        role: 'assistant',
        content: responseContent,
        timestamp: new Date().toISOString(),
      }

      const conv = await this.getConversation(nodeId)
      const userMsg: AiMessage = {
        id: `msg_${Date.now()}_user`,
        role: 'user',
        content: message,
        timestamp: new Date().toISOString(),
      }

      const messages = conv ? [...conv.messages, userMsg, aiMessage] : [userMsg, aiMessage]
      localStorage.setItem(
        `${CONVERSATION_PREFIX}${nodeId}`,
        JSON.stringify({ nodeId, messages }),
      )

      return aiMessage
    },

    async getConversation(nodeId: string): Promise<AiConversation | undefined> {
      const raw = localStorage.getItem(`${CONVERSATION_PREFIX}${nodeId}`)
      return raw ? (JSON.parse(raw) as AiConversation) : undefined
    },

    async clearConversation(nodeId: string): Promise<void> {
      localStorage.removeItem(`${CONVERSATION_PREFIX}${nodeId}`)
    },
  }
}
