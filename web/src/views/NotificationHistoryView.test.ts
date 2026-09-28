// @vitest-environment jsdom
import { mount, flushPromises } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import View from './NotificationHistoryView.vue'
import { ApiError } from '@/lib/api'

const mocks = vi.hoisted(() => ({ list: vi.fn() }))
vi.mock('@/lib/api', () => ({ ApiError: class extends Error { constructor(message: string, public status = 0) { super(message) } }, api: { cluster: { notificationHistory: mocks.list } } }))
vi.mock('@/i18n/phrase', () => ({ usePhraseCatalog: vi.fn(), phraseCatalogVersion: { value: 1 }, translatePhrase: (s: string) => s }))
vi.mock('vue-router', () => ({ useRoute: () => reactive({ query: {} }) }))
const wrappers: ReturnType<typeof mount>[] = []
const event = { id: '9', hostId: 'local', hostName: '本机', isLocal: true, rule: 'cpu', kind: 'alert', message: 'CPU 95% > 90%', delivery: 'local_only', attempts: 0, createdAt: '2026-09-28T01:00:00Z' }
const page = { items: [event], hosts: [{ id: 'local', name: '本机', isLocal: true }], nextCursor: '9', retentionDays: 30, maxEvents: 2000, maxBytes: 4194304 }
async function open() {
  const wrapper = mount(View, { global: { stubs: {
    LoadingState: { template: '<div>Loading</div>' },
    ErrorState: { props: ['message'], template: '<div role="alert">{{ message }}<button @click="$emit(\'retry\')">retry</button></div>' },
  } } })
  wrappers.push(wrapper); await flushPromises(); return wrapper
}
beforeEach(() => { vi.clearAllMocks(); mocks.list.mockResolvedValue(page) })
afterEach(() => wrappers.splice(0).forEach(wrapper => wrapper.unmount()))

describe('notification history', () => {
  it('identifies invalid filters without reporting a storage failure', async () => {
    mocks.list.mockRejectedValueOnce(new ApiError('invalid filters', 400))
    const wrapper = await open()
    expect(wrapper.text()).toContain('筛选条件无效')
    expect(wrapper.text()).not.toContain('数据目录')
  })
  it('filters on the server and paginates the applied query', async () => {
    const wrapper = await open()
    expect(wrapper.text()).toContain('仅本地')
    await wrapper.findAll('select')[1]!.setValue('local'); await flushPromises()
    expect(mocks.list).toHaveBeenLastCalledWith(expect.objectContaining({ host: 'local', since: expect.any(String) }), expect.any(AbortSignal))
    await wrapper.get('input[type="search"]').setValue('CPU')
    await wrapper.get('form').trigger('submit'); await flushPromises()
    mocks.list.mockResolvedValue({ ...page, items: [{ ...event, id: '8' }], nextCursor: '' })
    await wrapper.findAll('button').find(button => button.text() === '加载更多')!.trigger('click'); await flushPromises()
    expect(mocks.list).toHaveBeenLastCalledWith(expect.objectContaining({ host: 'local', search: 'CPU', cursor: '9' }), expect.any(AbortSignal))
    expect(wrapper.findAll('.notification-history__event')).toHaveLength(2)
  })
  it('shows empty and failed requests and allows retry', async () => {
    mocks.list.mockRejectedValueOnce(new Error('offline'))
    const wrapper = await open()
    expect(wrapper.text()).toContain('通知记录暂时不可用')
    mocks.list.mockResolvedValue({ ...page, items: [], nextCursor: '' })
    await wrapper.get('[role="alert"] button').trigger('click'); await flushPromises()
    expect(wrapper.text()).toContain('暂无符合条件的通知')
  })
  it('ignores an obsolete response after a filter change', async () => {
    let resolveFirst!: (value: typeof page) => void
    mocks.list.mockImplementationOnce(() => new Promise(resolve => { resolveFirst = resolve }))
    const wrapper = await open()
    mocks.list.mockResolvedValue({ ...page, items: [], nextCursor: '' })
    await wrapper.findAll('select')[1]!.setValue('local'); await flushPromises()
    resolveFirst(page); await flushPromises()
    expect(wrapper.findAll('.notification-history__event')).toHaveLength(0)
    expect(wrapper.text()).toContain('暂无符合条件的通知')
  })
})
