import { describe, it, expect } from 'vitest'

describe('测试基础设施', () => {
  it('基本断言应正常工作', () => {
    expect(1 + 1).toBe(2)
  })

  it('类型定义应正确导入', async () => {
    const types = await import('@/types')
    expect(types).toBeDefined()
  })
})
