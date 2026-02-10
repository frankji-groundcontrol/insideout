import { describe, it, expect } from 'vitest'
import { validateBundleModes, type BundleModes } from '@/services/registry'

describe('Bundle 配置校验', () => {
  it('全 mock 配置通过校验', () => {
    const modes: BundleModes = { auth: 'mock', data: 'mock', aiExport: 'mock' }
    expect(() => validateBundleModes(modes)).not.toThrow()
  })

  it('全 supabase 配置通过校验', () => {
    const modes: BundleModes = { auth: 'supabase', data: 'supabase', aiExport: 'supabase' }
    expect(() => validateBundleModes(modes)).not.toThrow()
  })

  it('auth=supabase + data=mock + aiExport=mock 通过校验', () => {
    const modes: BundleModes = { auth: 'supabase', data: 'mock', aiExport: 'mock' }
    expect(() => validateBundleModes(modes)).not.toThrow()
  })

  it('auth=supabase + data=supabase + aiExport=mock 通过校验', () => {
    const modes: BundleModes = { auth: 'supabase', data: 'supabase', aiExport: 'mock' }
    expect(() => validateBundleModes(modes)).not.toThrow()
  })

  it('data=supabase + auth=mock 抛出配置冲突错误', () => {
    const modes: BundleModes = { auth: 'mock', data: 'supabase', aiExport: 'mock' }
    expect(() => validateBundleModes(modes)).toThrow('data=supabase 要求 auth=supabase')
  })

  it('aiExport=supabase + auth=mock 抛出配置冲突错误', () => {
    const modes: BundleModes = { auth: 'mock', data: 'mock', aiExport: 'supabase' }
    expect(() => validateBundleModes(modes)).toThrow('aiExport=supabase 要求 auth=supabase')
  })

  it('无效的适配器模式值抛出错误', () => {
    const modes = { auth: 'invalid', data: 'mock', aiExport: 'mock' } as unknown as BundleModes
    expect(() => validateBundleModes(modes)).toThrow('配置值无效')
  })
})
