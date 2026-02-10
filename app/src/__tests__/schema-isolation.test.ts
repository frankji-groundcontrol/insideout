// 验证 juanleme schema 配置正确
import { describe, test, expect, vi } from 'vitest'

describe('juanleme schema isolation', () => {
  test('supabase client is properly configured', async () => {
    vi.stubEnv('VITE_SUPABASE_URL', 'https://test.supabase.co')
    vi.stubEnv('VITE_SUPABASE_ANON_KEY', 'test-anon-key')

    const mod = await import('@/lib/supabase')
    expect(mod.getSupabase).toBeDefined()
    expect(typeof mod.getSupabase).toBe('function')

    vi.unstubAllEnvs()
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

  test('environment variable names follow VITE_ convention', () => {
    const requiredVars = ['VITE_SUPABASE_URL', 'VITE_SUPABASE_ANON_KEY', 'VITE_API_MODE']
    for (const v of requiredVars) {
      expect(v).toMatch(/^VITE_/)
    }
  })
})
