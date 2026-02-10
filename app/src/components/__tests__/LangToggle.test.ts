import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import LangToggle from '@/components/common/LangToggle.vue'
import zhCN from '@/i18n/locales/zh-CN'
import enUS from '@/i18n/locales/en-US'

const createTestI18n = () =>
  createI18n({
    legacy: false,
    locale: 'zh-CN',
    fallbackLocale: 'zh-CN',
    messages: {
      'zh-CN': zhCN,
      'en-US': enUS,
    },
  })

describe('LangToggle', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('点击会在 zh-CN 与 en-US 之间切换', async () => {
    const i18n = createTestI18n()
    const wrapper = mount(LangToggle, {
      global: {
        plugins: [i18n],
      },
    })

    expect(i18n.global.locale.value).toBe('zh-CN')

    await wrapper.get('[data-testid="lang-toggle"]').trigger('click')
    expect(i18n.global.locale.value).toBe('en-US')

    await wrapper.get('[data-testid="lang-toggle"]').trigger('click')
    expect(i18n.global.locale.value).toBe('zh-CN')
  })

  it('会把语言偏好写入 localStorage', async () => {
    const wrapper = mount(LangToggle, {
      global: {
        plugins: [createTestI18n()],
      },
    })

    await wrapper.get('[data-testid="lang-toggle"]').trigger('click')
    expect(localStorage.getItem('juanleme-lang')).toBe('en-US')

    await wrapper.get('[data-testid="lang-toggle"]').trigger('click')
    expect(localStorage.getItem('juanleme-lang')).toBe('zh-CN')
  })
})
