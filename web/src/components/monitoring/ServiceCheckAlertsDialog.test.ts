// @vitest-environment jsdom
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, expect, it, vi } from 'vitest'
import ServiceCheckAlertsDialog from './ServiceCheckAlertsDialog.vue'
const mocks = vi.hoisted(() => ({ load: vi.fn(), save: vi.fn() }))
vi.mock('@/lib/api', () => ({ ApiError: class extends Error {}, api: { cluster: { serviceCheckAlerts: mocks.load, updateServiceCheckAlerts: mocks.save } } }))
vi.mock('@/i18n/phrase', () => ({ phraseCatalogVersion: { value: 0 }, translatePhrase: (v: string) => v, usePhraseCatalog: vi.fn() }))
const value = { enabled: false, repeat: false, subscriptions: [{ hostId: 'other', checkId: 'remote', revision: 'b' }], resourceVersion: 'v1', channelReady: false, pending: 0,
  hosts: [{ id: 'local', name: 'Local', state: 'online', checks: { available: true, intervalSeconds: 300, items: [{ id: 'health', revision: 'a', name: 'Health', target: 'example.com', kind: 'http', state: 'up' }] } }] }
beforeEach(() => { vi.clearAllMocks(); mocks.load.mockResolvedValue(structuredClone(value)); mocks.save.mockImplementation(async input => ({ ...value, ...input, resourceVersion: 'v2' })) })
function create() { return mount(ServiceCheckAlertsDialog, { props: { open: true, hostId: 'local' }, global: { stubs: { teleport: true } } }) }
it('saves the selected host without discarding hidden subscriptions', async () => {
  const wrapper = create(); await flushPromises()
  expect(wrapper.text()).toContain('请先在集群通知中启用')
  await wrapper.get('.service-alerts__check input').setValue(true)
  await wrapper.findAll('button').find(b => b.text() === '保存设置')!.trigger('click'); await flushPromises()
  expect(mocks.save).toHaveBeenCalledWith(expect.objectContaining({ expectedResourceVersion: 'v1', subscriptions: [value.subscriptions[0], { hostId: 'local', checkId: 'health', revision: 'a' }] }))
  expect(wrapper.text()).toContain('服务通知设置已保存。'); wrapper.unmount()
})
it('keeps unsaved selections after a failed save and supports a real load error', async () => {
  const wrapper = create(); await flushPromises()
  await wrapper.get('.service-alerts__check input').setValue(true); mocks.save.mockRejectedValueOnce(new Error('failed'))
  await wrapper.findAll('button').find(b => b.text() === '保存设置')!.trigger('click'); await flushPromises()
  expect(wrapper.get('.service-alerts__check input').element).toHaveProperty('checked', true)
  expect(wrapper.text()).toContain('保存失败'); wrapper.unmount()
  mocks.load.mockRejectedValueOnce(new Error('unavailable'))
  const failed = create(); await flushPromises(); expect(failed.text()).toContain('无法读取服务通知设置。'); failed.unmount()
})
