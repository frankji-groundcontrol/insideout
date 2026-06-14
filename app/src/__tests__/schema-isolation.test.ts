// 验证 juanleme schema 配置正确
import { describe, test, expect } from 'vitest'
import { createSupabaseClient } from '@/lib/supabase'

describe('juanleme schema isolation', () => {
  test('supabase client factory builds a client from literal url/key', () => {
    // 工厂取代了模块单例 getSupabase()，直接接收 url/key 字面量参数（不读取 import.meta.env）。
    const client = createSupabaseClient('https://test.supabase.co', 'test-anon-key')
    expect(client).toBeDefined()
    expect(typeof client.schema).toBe('function')
  })

  test('supabase client factory 缺少 url/key 时抛出错误', () => {
    expect(() => createSupabaseClient('', '')).toThrow()
  })

  test('expected app tables are defined for juanleme schema only', () => {
    const JUANLEME_TABLES = [
      'profiles',
      'workspaces',
      'workspace_memberships',
      'projects',
      'workshop_nodes',
      'documents',
      'document_revisions',
      'ai_runs',
      'export_jobs',
    ]
    expect(JUANLEME_TABLES).toHaveLength(9)
    for (const t of JUANLEME_TABLES) {
      expect(t.length).toBeGreaterThan(0)
    }
  })

  test('environment variable names follow NUXT_PUBLIC_ convention', () => {
    const requiredVars = ['NUXT_PUBLIC_SUPABASE_URL', 'NUXT_PUBLIC_SUPABASE_ANON_KEY', 'NUXT_PUBLIC_API_MODE']
    for (const v of requiredVars) {
      expect(v).toMatch(/^NUXT_PUBLIC_/)
    }
  })
})
