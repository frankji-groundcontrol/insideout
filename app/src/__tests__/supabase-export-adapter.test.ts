import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { IExportService } from '@/types/services'

const mockInvoke = vi.fn()

vi.mock('@/lib/supabase', () => ({
  getSupabase: () => ({
    functions: {
      invoke: mockInvoke,
    },
  }),
}))

describe('Supabase 导出适配器', () => {
  let service: IExportService

  beforeEach(async () => {
    vi.clearAllMocks()
    const mod = await import('@/services/supabase/exportService')
    service = mod.createSupabaseExportService()
  })

  it('generateMarkdown 使用 markdown 格式调用 Edge Function 并返回文本', async () => {
    mockInvoke.mockResolvedValue({
      data: {
        jobId: 'job-1',
        status: 'succeeded',
        format: 'markdown',
        content: '# 导出结果\n\n正文',
        createdAt: '2026-02-10T00:00:00Z',
      },
      error: null,
    })

    const markdown = await service.generateMarkdown({
      workshopId: 'ws-1',
      format: 'markdown',
      includeEmptyNodes: true,
    })

    expect(mockInvoke).toHaveBeenCalledWith('juanleme-export-document', {
      body: {
        workshopId: 'ws-1',
        format: 'markdown',
        includeEmptyNodes: true,
      },
    })
    expect(markdown).toContain('# 导出结果')
  })

  it('generatePrintHTML 使用 html 格式调用 Edge Function 并返回 HTML 字符串', async () => {
    mockInvoke.mockResolvedValue({
      data: {
        jobId: 'job-2',
        status: 'succeeded',
        format: 'html',
        content: '<!DOCTYPE html><html><body><h1>导出</h1></body></html>',
        createdAt: '2026-02-10T00:00:00Z',
      },
      error: null,
    })

    const html = await service.generatePrintHTML({
      workshopId: 'ws-2',
      format: 'print',
      includeEmptyNodes: false,
    })

    expect(mockInvoke).toHaveBeenCalledWith('juanleme-export-document', {
      body: {
        workshopId: 'ws-2',
        format: 'html',
        includeEmptyNodes: false,
      },
    })
    expect(html).toContain('<!DOCTYPE html>')
  })
})
