import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import ThemeToggle from '@/components/common/ThemeToggle.vue'
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

describe('ThemeToggle', () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.classList.remove('dark')
  })

  it('点击切换后会添加 dark 类', async () => {
    const wrapper = mount(ThemeToggle, {
      global: {
        plugins: [createTestI18n()],
      },
    })

    await wrapper.get('[data-testid="theme-toggle"]').trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('再次点击会移除 dark 类', async () => {
    const wrapper = mount(ThemeToggle, {
      global: {
        plugins: [createTestI18n()],
      },
    })

    await wrapper.get('[data-testid="theme-toggle"]').trigger('click')
    await wrapper.get('[data-testid="theme-toggle"]').trigger('click')

    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('会把主题偏好写入 localStorage', async () => {
    const wrapper = mount(ThemeToggle, {
      global: {
        plugins: [createTestI18n()],
      },
    })

    await wrapper.get('[data-testid="theme-toggle"]').trigger('click')
    expect(localStorage.getItem('juanleme-theme')).toBe('dark')

    await wrapper.get('[data-testid="theme-toggle"]').trigger('click')
    expect(localStorage.getItem('juanleme-theme')).toBe('light')
  })
})
