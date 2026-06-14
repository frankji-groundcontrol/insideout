import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { SupabaseClient } from '@supabase/supabase-js'
import type { IAiService } from '@/types/services'
import { AiRateLimitError, AiServiceUnavailableError } from '@/types'
import { createSupabaseAiService } from '@/services/supabase/aiService'

const mockInvoke = vi.fn()

// 注入式 stub 客户端：适配器工厂现在接收 SupabaseClient 参数。
function createStubClient(): SupabaseClient {
  return {
    functions: {
      invoke: mockInvoke,
    },
  } as unknown as SupabaseClient
}

describe('Supabase AI 适配器', () => {
  let service: IAiService

  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()

    service = createSupabaseAiService(createStubClient())
  })

  it('reply 会调用 Edge Function 并返回 AiMessage 结构', async () => {
    mockInvoke.mockResolvedValue({
      data: {
        id: 'run_001',
        role: 'assistant',
        content: '这是服务端回复',
        timestamp: '2026-02-10T00:00:00.000Z',
      },
      error: null,
    })

    const result = await service.reply('node_001', '你好')

    expect(mockInvoke).toHaveBeenCalledWith('juanleme-ai-generate', {
      body: { nodeId: 'node_001', message: '你好' },
    })
    expect(result).toEqual({
      id: 'run_001',
      role: 'assistant',
      content: '这是服务端回复',
      timestamp: '2026-02-10T00:00:00.000Z',
    })
  })

  it('reply 会把对话持久化到 localStorage', async () => {
    mockInvoke.mockResolvedValue({
      data: {
        id: 'run_002',
        role: 'assistant',
        content: '已记录你的想法',
        timestamp: '2026-02-10T00:00:01.000Z',
      },
      error: null,
    })

    await service.reply('node_abc', '请帮我整理')

    const raw = localStorage.getItem('ai-conv:node_abc')
    expect(raw).toBeTruthy()
    const parsed = JSON.parse(raw ?? '{}') as {
      nodeId: string
      messages: Array<{ role: string; content: string }>
    }

    expect(parsed.nodeId).toBe('node_abc')
    expect(parsed.messages).toHaveLength(2)
    expect(parsed.messages[0]?.role).toBe('user')
    expect(parsed.messages[0]?.content).toBe('请帮我整理')
    expect(parsed.messages[1]?.role).toBe('assistant')
    expect(parsed.messages[1]?.content).toBe('已记录你的想法')
  })

  it('getConversation 会从 localStorage 读取会话', async () => {
    localStorage.setItem(
      'ai-conv:node_read',
      JSON.stringify({
        nodeId: 'node_read',
        messages: [
          {
            id: 'm1',
            role: 'user',
            content: 'test',
            timestamp: '2026-02-10T00:00:02.000Z',
          },
        ],
      }),
    )

    const conv = await service.getConversation('node_read')

    expect(conv?.nodeId).toBe('node_read')
    expect(conv?.messages).toHaveLength(1)
    expect(conv?.messages[0]?.content).toBe('test')
  })

  it('clearConversation 会删除 localStorage 中的会话', async () => {
    localStorage.setItem('ai-conv:node_clear', JSON.stringify({ nodeId: 'node_clear', messages: [] }))

    await service.clearConversation('node_clear')

    expect(localStorage.getItem('ai-conv:node_clear')).toBeNull()
  })

  it('reply 在 429 时抛出 AiRateLimitError 并携带 retryAfterSeconds', async () => {
    mockInvoke.mockResolvedValue({
      data: {
        error: 'Rate limit exceeded',
        code: 'APP_THROTTLE',
        retry_after_seconds: 45,
        current_count: 10,
        max_requests: 10,
      },
      error: {
        message: 'Edge Function returned a non-2xx status code',
        context: {
          status: 429,
        },
      },
    })

    const promise = service.reply('node_429', 'hello')
    await expect(promise).rejects.toBeInstanceOf(AiRateLimitError)
    await expect(promise).rejects.toMatchObject({
      retryAfterSeconds: 45,
      currentCount: 10,
      maxRequests: 10,
    })
  })

  it('reply 在 503 时抛出 AiServiceUnavailableError', async () => {
    mockInvoke.mockResolvedValue({
      data: {
        error: 'AI service temporarily unavailable',
        code: 'CIRCUIT_OPEN',
        retry_after_seconds: 30,
        circuit_state: 'open',
      },
      error: {
        message: 'Edge Function returned a non-2xx status code',
        context: {
          status: 503,
        },
      },
    })

    const promise = service.reply('node_503', 'hello')
    await expect(promise).rejects.toBeInstanceOf(AiServiceUnavailableError)
    await expect(promise).rejects.toMatchObject({
      retryAfterSeconds: 30,
      circuitState: 'open',
    })
  })
})
