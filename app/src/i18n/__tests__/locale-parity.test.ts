import { describe, it, expect } from 'vitest'
import zhCN from '@/i18n/locales/zh-CN'
import enUS from '@/i18n/locales/en-US'

/** Flattens nested message objects into dotted key paths, e.g. "login.title". */
function flattenKeys(obj: Record<string, unknown>, prefix = ''): string[] {
  return Object.entries(obj).flatMap(([key, value]) => {
    const path = prefix ? `${prefix}.${key}` : key
    if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
      return flattenKeys(value as Record<string, unknown>, path)
    }
    return [path]
  })
}

describe('locale key parity', () => {
  it('zh-CN and en-US define exactly the same set of translation keys', () => {
    const zhKeys = new Set(flattenKeys(zhCN))
    const enKeys = new Set(flattenKeys(enUS))

    const missingInEn = [...zhKeys].filter((k) => !enKeys.has(k))
    const missingInZh = [...enKeys].filter((k) => !zhKeys.has(k))

    expect(missingInEn, `keys present in zh-CN but missing in en-US: ${missingInEn.join(', ')}`).toEqual([])
    expect(missingInZh, `keys present in en-US but missing in zh-CN: ${missingInZh.join(', ')}`).toEqual([])
  })
})
