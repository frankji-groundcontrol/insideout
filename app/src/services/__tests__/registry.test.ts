import { describe, it, expect, beforeEach } from 'vitest'
import { getServices } from '@/services/registry'

describe('ServiceRegistry', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  describe('workshop', () => {
    it('getWorkshops 返回工作坊数组', async () => {
      const { workshop } = getServices()
      const workshops = await workshop.getWorkshops()

      expect(Array.isArray(workshops)).toBe(true)
      expect(workshops.length).toBeGreaterThan(0)
      expect(workshops[0]).toHaveProperty('id')
      expect(workshops[0]).toHaveProperty('title')
    })

    it('saveSubmission 持久化后 getSubmission 可以读取', async () => {
      const { workshop } = getServices()
      const submission = {
        nodeId: 'node_test',
        userId: 'user_001',
        content: '测试提交内容',
        submittedAt: new Date().toISOString(),
      }

      const saved = await workshop.saveSubmission(submission)
      expect(saved).toEqual(submission)

      const loaded = await workshop.getSubmission('node_test')
      expect(loaded).toEqual(submission)
    })
  })

  describe('editor', () => {
    it('saveDraft 后 loadDraft 可以正确读取', async () => {
      const { editor } = getServices()
      const draft = {
        key: 'test-draft-key',
        content: { text: '草稿内容' },
        updatedAt: new Date().toISOString(),
        revision: 1,
      }

      await editor.saveDraft(draft)
      const loaded = await editor.loadDraft('test-draft-key')

      expect(loaded).toEqual(draft)
    })

    it('deleteDraft 后 loadDraft 返回 undefined', async () => {
      const { editor } = getServices()
      const draft = {
        key: 'to-delete',
        content: 'temp',
        updatedAt: new Date().toISOString(),
        revision: 1,
      }

      await editor.saveDraft(draft)
      await editor.deleteDraft('to-delete')
      const loaded = await editor.loadDraft('to-delete')

      expect(loaded).toBeUndefined()
    })
  })

  describe('ai', () => {
    it('reply 对包含产品关键词的消息返回非空 AiMessage', async () => {
      const { ai } = getServices()
      const msg = await ai.reply('node_1', '产品灵感')

      expect(msg).toHaveProperty('id')
      expect(msg.role).toBe('assistant')
      expect(msg.content.length).toBeGreaterThan(0)
      expect(msg).toHaveProperty('timestamp')
    })

    it('reply 对无匹配关键词的消息返回默认回复', async () => {
      const { ai } = getServices()
      const msg = await ai.reply('node_1', 'hello random text')

      expect(msg.role).toBe('assistant')
      expect(msg.content).toContain('加油')
    })
  })

  describe('export', () => {
    it('generateMarkdown 返回包含工作坊标题的 markdown 字符串', async () => {
      const { export: exportService } = getServices()
      const md = await exportService.generateMarkdown({
        workshopId: 'ws_001',
        format: 'markdown',
        includeEmptyNodes: true,
      })

      expect(typeof md).toBe('string')
      expect(md).toContain('# ')
      expect(md).toContain('周三下午搓个垃圾出来')
    })

    it('generatePrintHTML 返回有效的 HTML 文档', async () => {
      const { export: exportService } = getServices()
      const html = await exportService.generatePrintHTML({
        workshopId: 'ws_001',
        format: 'print',
        includeEmptyNodes: false,
      })

      expect(html).toContain('<!DOCTYPE html>')
      expect(html).toContain('<html')
      expect(html).toContain('</html>')
    })
  })
})
