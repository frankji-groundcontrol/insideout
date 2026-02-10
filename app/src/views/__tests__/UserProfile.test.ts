import { beforeEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import UserProfile from '@/views/profile/UserProfile.vue'
import { useUserStore } from '@/stores/user'
import zhCN from '@/i18n/locales/zh-CN'
import enUS from '@/i18n/locales/en-US'
import type { UserProfile as UserProfileType } from '@/types'

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

const createMockUser = (): UserProfileType => ({
  id: 'user_1',
  email: 'tester@example.com',
  username: 'tester',
  avatar_url: 'https://example.com/avatar.png',
  bio: 'hello bio',
  keywords: ['product', 'design'],
  role: 'user',
  created_at: '2026-02-10T00:00:00.000Z',
})

describe('UserProfile', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('表单会回显 userStore 中的用户名、邮箱和简介', () => {
    const userStore = useUserStore()
    userStore.user = createMockUser()

    const wrapper = mount(UserProfile, {
      global: {
        plugins: [createTestI18n()],
      },
    })

    const usernameInput = wrapper.get('[data-testid="profile-username"]')
    const emailInput = wrapper.get('[data-testid="profile-email"]')
    const bioInput = wrapper.get('[data-testid="profile-bio"]')

    expect((usernameInput.element as HTMLInputElement).value).toBe('tester')
    expect((emailInput.element as HTMLInputElement).value).toBe('tester@example.com')
    expect((bioInput.element as HTMLTextAreaElement).value).toBe('hello bio')
  })

  it('点击保存后会更新 userStore 中的资料', async () => {
    const userStore = useUserStore()
    userStore.user = createMockUser()

    const wrapper = mount(UserProfile, {
      global: {
        plugins: [createTestI18n()],
      },
    })

    await wrapper.get('[data-testid="profile-username"]').setValue('new-name')
    await wrapper.get('[data-testid="profile-bio"]').setValue('new bio')
    await wrapper.get('[data-testid="profile-keywords"]').setValue('frontend,vue,ts')
    await wrapper.get('[data-testid="profile-form"]').trigger('submit')

    expect(userStore.user?.username).toBe('new-name')
    expect(userStore.user?.bio).toBe('new bio')
    expect(userStore.user?.keywords).toEqual(['frontend', 'vue', 'ts'])
    expect(wrapper.text()).toContain('保存成功')
  })

  it('用户名为空时会显示校验错误', async () => {
    const userStore = useUserStore()
    userStore.user = createMockUser()

    const wrapper = mount(UserProfile, {
      global: {
        plugins: [createTestI18n()],
      },
    })

    await wrapper.get('[data-testid="profile-username"]').setValue('')
    await wrapper.get('[data-testid="profile-form"]').trigger('submit')

    expect(wrapper.text()).toContain('用户名不能为空')
  })

  it('updateProfile 会合并用户字段并持久化到 localStorage', () => {
    const userStore = useUserStore()
    userStore.user = createMockUser()

    userStore.updateProfile({ username: 'merged-name', keywords: ['ai', 'workflow'] })

    expect(userStore.user?.username).toBe('merged-name')
    expect(userStore.user?.keywords).toEqual(['ai', 'workflow'])

    const savedUser = JSON.parse(localStorage.getItem('juanleme-user') ?? '{}') as UserProfileType
    expect(savedUser.username).toBe('merged-name')
    expect(savedUser.keywords).toEqual(['ai', 'workflow'])
  })
})
