import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { AdminUser } from '@/types'
import UsersView from '../UsersView.vue'

const {
  listUsers,
  getAllGroups,
  getBatchUsersUsage,
  listEnabledDefinitions,
  getBatchUserAttributes,
  deleteUser,
  showError,
  showSuccess
} = vi.hoisted(() => ({
  listUsers: vi.fn(),
  getAllGroups: vi.fn(),
  getBatchUsersUsage: vi.fn(),
  listEnabledDefinitions: vi.fn(),
  getBatchUserAttributes: vi.fn(),
  deleteUser: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      list: listUsers,
      toggleStatus: vi.fn(),
      delete: deleteUser
    },
    groups: {
      getAll: getAllGroups
    },
    dashboard: {
      getBatchUsersUsage
    },
    userAttributes: {
      listEnabledDefinitions,
      getBatchUserAttributes
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: { count?: number }) => params?.count === undefined ? key : `${key}:${params.count}`
    })
  }
})

const createAdminUser = (overrides: Partial<AdminUser> = {}): AdminUser => ({
  id: 42,
  username: 'scoped-user',
  email: 'scoped@example.com',
  role: 'user',
  balance: 0,
  concurrency: 1,
  status: 'active',
  allowed_groups: [],
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-04-17T00:00:00Z',
  updated_at: '2026-04-17T00:00:00Z',
  notes: '',
  last_active_at: '2026-04-16T02:00:00Z',
  last_used_at: '2026-04-17T02:00:00Z',
  current_concurrency: 0,
  ...overrides
})

const mountUsersView = () => mount(UsersView, {
  global: {
    stubs: {
      AppLayout: { template: '<div><slot /></div>' },
      TablePageLayout: {
        template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
      },
      DataTable: DataTableStub,
      Pagination: true,
      ConfirmDialog: { props: ['show', 'message'], emits: ['confirm', 'cancel'], template: '<div v-if="show"><span>{{ message }}</span><button data-test="confirm-delete" @click="$emit(\'confirm\')">confirm</button><button data-test="cancel-delete" @click="$emit(\'cancel\')">cancel</button></div>' },
      EmptyState: true,
      GroupBadge: true,
      Select: true,
      UserAttributesConfigModal: true,
      UserConcurrencyCell: true,
      PlatformUsageBreakdown: { props: ['today', 'total', 'byPlatform'], template: '<div data-test="usage-cell">{{ total }}</div>' },
      PlatformCostCell: { props: ['usage'], template: '<div data-test="platform-cost">{{ usage?.total_actual_cost ?? "-" }}</div>' },
      UserCreateModal: true,
      UserEditModal: true,
      BulkEditUserModal: BulkEditUserModalStub,
      UserPlatformQuotaModal: true,
      UserApiKeysModal: true,
      UserAllowedGroupsModal: true,
      UserBalanceModal: true,
      UserBalanceHistoryModal: true,
      GroupReplaceModal: true,
      Icon: true,
      Teleport: true
    }
  }
})

const DataTableStub = {
  props: ['columns', 'data', 'selectedKeys'],
  emits: ['sort', 'update:selectedKeys'],
  template: `
    <div>
      <div data-test="columns">{{ columns.map(col => col.key).join(',') }}</div>
      <div data-test="row-order">{{ data.map(row => row.email).join(',') }}</div>
      <div data-test="selected-keys">{{ (selectedKeys || []).join(',') }}</div>
      <button data-test="sort-last-used" @click="$emit('sort', 'last_used_at', 'desc')">sort</button>
      <button
        v-for="row in data"
        :key="'select-' + row.id"
        :data-test="'select-' + row.id"
        @click="$emit('update:selectedKeys', Array.from(new Set([...(selectedKeys || []), row.id])))"
      >
        select
      </button>
      <template v-for="col in columns" :key="col.key">
        <slot :name="'header-' + col.key" :column="col" />
      </template>
      <div v-for="row in data" :key="row.id">
        <div data-test="row-email">{{ row.email }}</div>
        <slot name="cell-last_used_at" :value="row.last_used_at" :row="row" />
        <slot v-for="column in columns" :name="'cell-' + column.key" :value="row[column.key]" :row="row" />
      </div>
    </div>
  `
}

const PaginationStub = {
  emits: ['update:page'],
  template: '<button data-test="next-page" @click="$emit(\'update:page\', 2)">next</button>'
}

const BulkEditUserModalStub = {
  props: ['show', 'selectedIds'],
  emits: ['close', 'success'],
  template: `
    <div v-if="show" data-test="bulk-modal">
      <span data-test="bulk-modal-ids">{{ selectedIds.join(',') }}</span>
      <button data-test="bulk-success" @click="$emit('success', selectedIds.length)">success</button>
    </div>
  `
}

describe('admin UsersView', () => {
  beforeEach(() => {
    vi.useRealTimers()
    localStorage.clear()

    listUsers.mockReset()
    deleteUser.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    getAllGroups.mockReset()
    getBatchUsersUsage.mockReset()
    listEnabledDefinitions.mockReset()
    getBatchUserAttributes.mockReset()

    listUsers.mockResolvedValue({
      items: [createAdminUser()],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getAllGroups.mockResolvedValue([])
    getBatchUsersUsage.mockResolvedValue({ stats: {} })
    listEnabledDefinitions.mockResolvedValue([])
    getBatchUserAttributes.mockResolvedValue({ values: {} })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('confirms bulk user deletion and keeps failed IDs selected', async () => {
    deleteUser.mockImplementation(async (id: number) => { if (id === 43) throw new Error('cannot delete') })
    listUsers.mockResolvedValue({ items: [createAdminUser({ id: 42 }), createAdminUser({ id: 43 })], total: 2, page: 1, page_size: 20, pages: 1 })
    const wrapper = mountUsersView()
    await flushPromises()
    await wrapper.get('[data-test="select-42"]').trigger('click')
    await wrapper.get('[data-test="select-43"]').trigger('click')
    await wrapper.get('[data-test="bulk-delete-users"]').trigger('click')
    expect(deleteUser).not.toHaveBeenCalled()
    await wrapper.get('[data-test="confirm-delete"]').trigger('click')
    await flushPromises()
    expect(deleteUser.mock.calls).toEqual([[42], [43]])
    expect(wrapper.get('[data-test="selected-keys"]').text()).toBe('43')
    expect(showSuccess).toHaveBeenCalledWith('admin.users.bulkDelete.success:1')
    expect(showError).toHaveBeenCalledWith('admin.users.bulkDelete.failed:1')
    wrapper.unmount()
  })

  it('取消批量删除不调用接口且保留选择', async () => {
    const wrapper = mountUsersView()
    await flushPromises()
    await wrapper.get('[data-test="select-42"]').trigger('click')
    await wrapper.get('[data-test="bulk-delete-users"]').trigger('click')
    await wrapper.get('[data-test="cancel-delete"]').trigger('click')
    expect(deleteUser).not.toHaveBeenCalled()
    expect(wrapper.get('[data-test="selected-keys"]').text()).toBe('42')
    wrapper.unmount()
  })

  it('删除只处理确认快照，不删除操作期间新选中的用户', async () => {
    listUsers.mockResolvedValue({ items: [createAdminUser({ id: 42 }), createAdminUser({ id: 43 })], total: 2, page: 1, page_size: 20, pages: 1 })
    let finish!: () => void
    deleteUser.mockReturnValue(new Promise<void>(resolve => { finish = resolve }))
    const wrapper = mountUsersView()
    await flushPromises()
    await wrapper.get('[data-test="select-42"]').trigger('click')
    await wrapper.get('[data-test="bulk-delete-users"]').trigger('click')
    await wrapper.get('[data-test="confirm-delete"]').trigger('click')
    expect(wrapper.get('[data-test="bulk-delete-users"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-test="select-43"]').trigger('click')
    finish()
    await flushPromises()
    expect(deleteUser.mock.calls).toEqual([[42]])
    expect(wrapper.get('[data-test="selected-keys"]').text()).toBe('43')
    wrapper.unmount()
  })

  it('shows active, used, and created activity columns in order and requests last_used_at sort', async () => {
    const wrapper = mountUsersView()

    await flushPromises()

    const columns = wrapper.get('[data-test="columns"]').text()
    const visibleColumns = columns.split(',')
    expect(visibleColumns.slice(-4, -1)).toEqual(['last_active_at', 'last_used_at', 'created_at'])
    expect(visibleColumns).not.toContain('last_login_at')

    await wrapper.get('[data-test="sort-last-used"]').trigger('click')
    await flushPromises()

    expect(listUsers).toHaveBeenLastCalledWith(
      1,
      20,
      expect.objectContaining({
        sort_by: 'last_used_at',
        sort_order: 'desc'
      }),
      expect.any(Object)
    )
  })

  it('hides new platform usage columns for existing column settings schema', async () => {
    localStorage.setItem('user-hidden-columns', JSON.stringify(['notes']))
    localStorage.setItem('user-column-settings-version', '1')

    const wrapper = mountUsersView()
    await flushPromises()

    const visibleColumns = wrapper.get('[data-test="columns"]').text().split(',')
    expect(visibleColumns).not.toContain('usage_anthropic')
    expect(visibleColumns).not.toContain('usage_openai')
    expect(visibleColumns).not.toContain('usage_gemini')
    expect(visibleColumns).not.toContain('usage_antigravity')
    expect(visibleColumns).not.toContain('usage_qoder')
    expect(JSON.parse(localStorage.getItem('user-hidden-columns') || '[]')).toEqual(
      expect.arrayContaining(['usage_anthropic', 'usage_openai', 'usage_gemini', 'usage_antigravity', 'usage_qoder'])
    )
  })

  it('sorts current page by platform usage after enabling a platform column', async () => {
    const highUsageUser = createAdminUser()
    highUsageUser.id = 42
    highUsageUser.email = 'high@example.com'
    const lowUsageUser = createAdminUser()
    lowUsageUser.id = 7
    lowUsageUser.email = 'low@example.com'

    localStorage.setItem(
      'user-hidden-columns',
      JSON.stringify(['notes', 'groups', 'subscriptions', 'usage', 'concurrency', 'usage_openai', 'usage_gemini', 'usage_antigravity', 'usage_qoder'])
    )
    localStorage.setItem('user-column-settings-version', '2')
    localStorage.setItem(
      'admin-users-usage-sort',
      JSON.stringify({ key: 'usage_anthropic', metric: 'total', order: 'desc' })
    )
    listUsers.mockResolvedValue({
      items: [lowUsageUser, highUsageUser],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getBatchUsersUsage.mockResolvedValue({
      stats: {
        42: {
          user_id: 42,
          today_actual_cost: 1,
          total_actual_cost: 10,
          by_platform: [{ platform: 'anthropic', today_actual_cost: 1, total_actual_cost: 10 }]
        },
        7: {
          user_id: 7,
          today_actual_cost: 1,
          total_actual_cost: 1,
          by_platform: [{ platform: 'anthropic', today_actual_cost: 1, total_actual_cost: 1 }]
        }
      }
    })

    const wrapper = mountUsersView()
    await flushPromises()
    await new Promise(resolve => window.setTimeout(resolve, 60))
    await flushPromises()

    expect(getBatchUsersUsage).toHaveBeenCalledWith([7, 42])
    expect(wrapper.findAll('[data-test="row-email"]').map(node => node.text())).toEqual([
      'high@example.com',
      'low@example.com'
    ])
  })

  it('clears usage current-page sort when switching to last_used_at server sort', async () => {
    vi.useFakeTimers()
    localStorage.setItem('user-column-settings-version', '4')
    localStorage.setItem(
      'user-hidden-columns',
      JSON.stringify([
        'notes',
        'groups',
        'subscriptions',
        'concurrency',
        'usage_anthropic',
        'usage_openai',
        'usage_gemini',
        'usage_antigravity',
        'usage_qoder',
        'balance_platform_quota'
      ])
    )

    listUsers.mockResolvedValue({
      items: [
        createAdminUser({ id: 1, email: 'last-used-first@example.com' }),
        createAdminUser({ id: 2, email: 'usage-first@example.com' })
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getBatchUsersUsage.mockResolvedValue({
      stats: {
        1: { user_id: 1, today_actual_cost: 1, total_actual_cost: 1, by_platform: [] },
        2: { user_id: 2, today_actual_cost: 9, total_actual_cost: 9, by_platform: [] }
      }
    })

    const wrapper = mountUsersView()
    await flushPromises()
    await vi.advanceTimersByTimeAsync(50)
    await flushPromises()

    expect(wrapper.get('[data-test="row-order"]').text()).toBe('last-used-first@example.com,usage-first@example.com')

    await wrapper.get('[data-test="usage-sort-trigger-usage"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="usage-sort-usage-today"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="row-order"]').text()).toBe('usage-first@example.com,last-used-first@example.com')
    expect(localStorage.getItem('admin-users-usage-sort')).toContain('"key":"usage"')

    await wrapper.get('[data-test="sort-last-used"]').trigger('click')
    await flushPromises()

    expect(localStorage.getItem('admin-users-usage-sort')).toBeNull()
    expect(wrapper.get('[data-test="row-order"]').text()).toBe('last-used-first@example.com,usage-first@example.com')
    expect(listUsers).toHaveBeenLastCalledWith(
      1,
      20,
      expect.objectContaining({
        sort_by: 'last_used_at',
        sort_order: 'desc'
      }),
      expect.any(Object)
    )
  })

  it('keeps selected user IDs across pages and clears them after a successful bulk update', async () => {
    let refreshed = false
    listUsers.mockImplementation(async (page: number) => {
      const user = page === 2
        ? createAdminUser({
            id: 43,
            email: refreshed ? 'refreshed-page-two@example.com' : 'page-two@example.com'
          })
        : createAdminUser({ id: 42, email: 'page-one@example.com' })
      return {
        items: [user],
        total: 2,
        page,
        page_size: 20,
        pages: 2
      }
    })

    const wrapper = mount(UsersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: PaginationStub,
          ConfirmDialog: true,
          EmptyState: true,
          GroupBadge: true,
          Select: true,
          UserAttributesConfigModal: true,
          UserConcurrencyCell: true,
          UserCreateModal: true,
          UserEditModal: true,
          BulkEditUserModal: BulkEditUserModalStub,
          UserPlatformQuotaModal: true,
          UserApiKeysModal: true,
          UserAllowedGroupsModal: true,
          UserBalanceModal: true,
          UserBalanceHistoryModal: true,
          GroupReplaceModal: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.find('[data-test="bulk-edit-limits"]').exists()).toBe(false)
    await wrapper.get('[data-test="select-42"]').trigger('click')
    expect(wrapper.get('[data-test="selected-keys"]').text()).toBe('42')
    expect(wrapper.find('[data-test="bulk-edit-limits"]').exists()).toBe(true)

    await wrapper.get('[data-test="next-page"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-test="selected-keys"]').text()).toBe('42')

    await wrapper.get('[data-test="select-43"]').trigger('click')
    expect(wrapper.get('[data-test="selected-keys"]').text()).toBe('42,43')

    await wrapper.get('[data-test="bulk-edit-limits"]').trigger('click')
    expect(wrapper.get('[data-test="bulk-modal-ids"]').text()).toBe('42,43')

    const callsBeforeSuccess = listUsers.mock.calls.length
    refreshed = true
    await wrapper.get('[data-test="bulk-success"]').trigger('click')
    await flushPromises()

    expect(listUsers.mock.calls.length).toBeGreaterThan(callsBeforeSuccess)
    expect(wrapper.get('[data-test="row-order"]').text()).toBe('refreshed-page-two@example.com')
    expect(wrapper.find('[data-test="bulk-edit-limits"]').exists()).toBe(false)
    expect(wrapper.get('[data-test="selected-keys"]').text()).toBe('')
  })
})
