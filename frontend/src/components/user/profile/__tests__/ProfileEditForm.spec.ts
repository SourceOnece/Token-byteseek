import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProfileEditForm from '@/components/user/profile/ProfileEditForm.vue'
import type { User } from '@/types'

const {
  updateProfileMock,
  showSuccessMock,
  showErrorMock,
  authStoreState,
} = vi.hoisted(() => ({
  updateProfileMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showErrorMock: vi.fn(),
  authStoreState: {
    user: null as User | null,
  },
}))

vi.mock('@/api', () => ({
  userAPI: {
    updateProfile: updateProfileMock,
  },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStoreState,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: showSuccessMock,
    showError: showErrorMock,
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        if (key === 'profile.editProfile') return 'Edit profile'
        if (key === 'profile.username') return 'Username'
        if (key === 'profile.enterUsername') return 'Enter username'
        if (key === 'profile.updateProfile') return 'Update profile'
        if (key === 'profile.updating') return 'Updating...'
        if (key === 'profile.updateSuccess') return 'Profile updated'
        if (key === 'profile.updateFailed') return 'Update failed'
        if (key === 'profile.usernameRequired') return 'Username is required'
        return key
      },
    }),
  }
})

function createUser(overrides: Partial<User> = {}): User {
  return {
    id: 5,
    username: 'alice-new',
    email: 'alice@example.com',
    role: 'user',
    balance: 10,
    concurrency: 2,
    status: 'active',
    allowed_groups: null,
    balance_notify_enabled: true,
    balance_notify_threshold: null,
    balance_notify_extra_emails: [],
    created_at: '2026-04-20T00:00:00Z',
    updated_at: '2026-04-20T00:00:00Z',
    ...overrides,
  }
}

describe('ProfileEditForm', () => {
  // 原有成功更新断言继续保留，补结构化 API 错误与旧 detail 回退。
  it.each([
    [{ status: 400, message: 'username is too long' }, 'username is too long'],
    [{ response: { data: { detail: 'backend failure' } } }, 'backend failure'],
    [{}, 'Update failed'],
  ])('保留失败前的用户状态 %j', async (error, expectedMessage) => {
    updateProfileMock.mockRejectedValue(error)
    authStoreState.user = createUser({ username: 'alice' })
    const wrapper = mount(ProfileEditForm, { props: { initialUsername: 'alice' } })
    await wrapper.get('#username').setValue('new-name')
    await wrapper.get('form').trigger('submit')
    expect(showErrorMock).toHaveBeenLastCalledWith(expectedMessage)
    expect(authStoreState.user.username).toBe('alice')
    wrapper.unmount()
  })

  beforeEach(() => {
    updateProfileMock.mockReset()
    showSuccessMock.mockReset()
    showErrorMock.mockReset()
    authStoreState.user = null
  })

  it('updates username without sending email', async () => {
    const updatedUser = createUser()
    updateProfileMock.mockResolvedValue(updatedUser)

    const wrapper = mount(ProfileEditForm, {
      props: {
        initialUsername: 'alice',
      },
    })

    await wrapper.get('#username').setValue('alice-new')
    await wrapper.get('form').trigger('submit')

    expect(updateProfileMock).toHaveBeenCalledWith({ username: 'alice-new' })
    expect(authStoreState.user).toEqual(updatedUser)
    expect(showSuccessMock).toHaveBeenCalledWith('Profile updated')
    expect(showErrorMock).not.toHaveBeenCalled()
  })
})
