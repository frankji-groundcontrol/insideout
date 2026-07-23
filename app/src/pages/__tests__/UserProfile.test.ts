import { beforeEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import zhCN from '@/i18n/locales/zh-CN'
import enUS from '@/i18n/locales/en-US'
import type { UserProfile as UserProfileType } from '@/types'

// Only the pure client-side validation is tested here — it runs and
// short-circuits before any service call, so it needs no backend and no
// mock. Save/update behavior now requires a real Go server; see
// docs/plans/2026-07-20-go-rewrite/02-backend-go.md §5 for the
// integration-test plan once DATABASE_URL is available.

import UserProfile from '@/pages/profile.vue'
import { useUserStore } from '@/stores/user'

const createTestI18n = () =>
  createI18n({
    legacy: false,
    locale: 'zh-CN',
    fallbackLocale: 'zh-CN',
    messages: { 'zh-CN': zhCN, 'en-US': enUS },
  })

const createMockUser = (): UserProfileType => ({
  id: 'user_1',
  email: 'tester@example.com',
  username: 'tester',
  avatarUrl: 'https://example.com/avatar.png',
  bio: 'hello bio',
  keywords: ['product', 'design'],
})

describe('UserProfile', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('表单会回显 userStore 中的用户名、邮箱和简介', () => {
    const userStore = useUserStore()
    userStore.user = createMockUser()

    const wrapper = mount(UserProfile, { global: { plugins: [createTestI18n()] } })

    expect((wrapper.get('[data-testid="profile-username"]').element as HTMLInputElement).value).toBe('tester')
    expect((wrapper.get('[data-testid="profile-email"]').element as HTMLInputElement).value).toBe(
      'tester@example.com',
    )
    expect((wrapper.get('[data-testid="profile-bio"]').element as HTMLTextAreaElement).value).toBe('hello bio')
  })

  it('用户名为空时会显示校验错误（不发起任何服务调用）', async () => {
    const userStore = useUserStore()
    userStore.user = createMockUser()

    const wrapper = mount(UserProfile, { global: { plugins: [createTestI18n()] } })

    await wrapper.get('[data-testid="profile-username"]').setValue('')
    await wrapper.get('[data-testid="profile-form"]').trigger('submit')

    expect(wrapper.text()).toContain('用户名不能为空')
  })
})
