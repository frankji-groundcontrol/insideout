import type { SupabaseClient } from '@supabase/supabase-js'
import type { ExportConfig } from '@/types'
import type { IExportService } from '@/types/services'

type ExportFunctionFormat = 'markdown' | 'html'

type ExportFunctionPayload = {
  workshopId: string
  format: ExportFunctionFormat
  includeEmptyNodes: boolean
}

type ExportFunctionResponse = {
  jobId: string
  status: string
  format: ExportFunctionFormat
  downloadUrl?: string | null
  content?: string
  createdAt: string
}

async function invokeExportFunction(
  supabase: SupabaseClient,
  payload: ExportFunctionPayload,
): Promise<ExportFunctionResponse> {
  const { data, error } = await supabase.functions.invoke('juanleme-export-document', {
    body: payload,
  })

  if (error) {
    throw new Error(error.message)
  }

  if (!data || typeof data !== 'object') {
    throw new Error('导出函数返回了无效响应')
  }

  return data as ExportFunctionResponse
}

async function resolveExportContent(response: ExportFunctionResponse): Promise<string> {
  // 优先使用函数内联内容，减少一次网络请求。
  if (typeof response.content === 'string') {
    return response.content
  }

  if (!response.downloadUrl) {
    throw new Error('导出响应缺少下载地址')
  }

  // 当函数仅返回签名 URL 时，前端拉取真实文档文本。
  const result = await fetch(response.downloadUrl)
  if (!result.ok) {
    throw new Error(`下载导出内容失败：${result.status}`)
  }

  return result.text()
}

export function createSupabaseExportService(
  supabase: SupabaseClient,
): IExportService {
  return {
    async generateMarkdown(config: ExportConfig): Promise<string> {
      const response = await invokeExportFunction(supabase, {
        workshopId: config.workshopId,
        format: 'markdown',
        includeEmptyNodes: config.includeEmptyNodes,
      })

      return resolveExportContent(response)
    },

    async generatePrintHTML(config: ExportConfig): Promise<string> {
      const response = await invokeExportFunction(supabase, {
        workshopId: config.workshopId,
        format: 'html',
        includeEmptyNodes: config.includeEmptyNodes,
      })

      return resolveExportContent(response)
    },
  }
}
