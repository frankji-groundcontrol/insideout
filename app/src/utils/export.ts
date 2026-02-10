import { nextTick } from 'vue'
import type { RoadmapNode, Workshop } from '@/types'

function normalizeContent(content?: string): string {
  const trimmed = content?.trim()
  return trimmed && trimmed.length > 0 ? trimmed : '[未完成]'
}

export function generateMarkdown(workshop: Workshop, nodes: RoadmapNode[]): string {
  const lines: string[] = [
    `# ${workshop.title}`,
    '',
    workshop.description,
    '',
    '---',
    '',
  ]

  for (const node of nodes) {
    lines.push(`## ${node.order}. ${node.title}`)
    lines.push('')
    lines.push(`> ${node.description}`)
    lines.push('')
    lines.push(normalizeContent(node.content))
    lines.push('')
  }

  return lines.join('\n')
}

export function downloadMarkdown(content: string, filename: string): void {
  const blob = new Blob([content], { type: 'text/markdown' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')

  link.href = url
  link.download = filename
  link.style.display = 'none'
  document.body.append(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

export function triggerPrint(): void {
  void nextTick().then(() => {
    window.print()
  })
}
