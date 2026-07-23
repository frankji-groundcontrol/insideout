import { marked } from 'marked'
import DOMPurify from 'dompurify'

marked.setOptions({
  gfm: true,
  // Chat-style line handling: a single newline becomes a line break, matching
  // how people actually write in a conversation box.
  breaks: true,
})

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

/**
 * Render a conversation message's markdown to sanitized, v-html-safe markup.
 *
 * Sanitization needs a DOM. Coach/user messages only ever render client-side
 * (history loads in onMounted, streaming is interactive), so the DOMPurify path
 * is the one that runs in practice. On the server DOMPurify reports
 * unsupported; we degrade to escaped plain text rather than trust raw HTML.
 */
export function renderMarkdown(src: string): string {
  const input = src ?? ''
  if (!DOMPurify.isSupported) {
    return escapeHtml(input).replace(/\n/g, '<br>')
  }
  const html = marked.parse(input, { async: false }) as string
  return DOMPurify.sanitize(html, { USE_PROFILES: { html: true } })
}
