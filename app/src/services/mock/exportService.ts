import type { IExportService } from '@/types/services'
import type { ExportConfig } from '@/types'
import { MOCK_WORKSHOPS, MOCK_NODES } from './data'

const delay = (ms = 300) =>
  new Promise(resolve => setTimeout(resolve, ms + Math.random() * 200))

export function createMockExportService(): IExportService {
  return {
    async generateMarkdown(config: ExportConfig): Promise<string> {
      await delay()

      const workshop = MOCK_WORKSHOPS.find(w => w.id === config.workshopId)
      if (!workshop) throw new Error(`Workshop not found: ${config.workshopId}`)

      const nodes = MOCK_NODES.filter(n => n.workshop_id === config.workshopId)
      const filtered = config.includeEmptyNodes
        ? nodes
        : nodes.filter(n => n.content)

      const lines: string[] = [
        `# ${workshop.title}`,
        '',
        workshop.description,
        '',
        `---`,
        '',
      ]

      for (const node of filtered) {
        lines.push(`## ${node.order}. ${node.title}`)
        lines.push('')
        lines.push(`> ${node.description}`)
        lines.push('')
        if (node.content) {
          lines.push(node.content)
          lines.push('')
        }
      }

      return lines.join('\n')
    },

    async generatePrintHTML(config: ExportConfig): Promise<string> {
      await delay()

      const markdown = await this.generateMarkdown(config)
      const workshop = MOCK_WORKSHOPS.find(w => w.id === config.workshopId)
      const title = workshop?.title || 'Workshop Export'

      return [
        '<!DOCTYPE html>',
        '<html lang="zh-CN">',
        '<head>',
        `  <meta charset="UTF-8">`,
        `  <title>${title}</title>`,
        '  <style>',
        '    body { font-family: -apple-system, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; color: #333; }',
        '    h1 { border-bottom: 2px solid #333; padding-bottom: 0.5rem; }',
        '    h2 { color: #555; margin-top: 2rem; }',
        '    blockquote { border-left: 3px solid #ccc; padding-left: 1rem; color: #666; }',
        '    @media print { body { padding: 0; } }',
        '  </style>',
        '</head>',
        '<body>',
        `  <pre style="white-space: pre-wrap;">${markdown}</pre>`,
        '</body>',
        '</html>',
      ].join('\n')
    },
  }
}
