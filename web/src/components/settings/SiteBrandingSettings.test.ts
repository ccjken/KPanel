// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
const mocks = vi.hoisted(() => ({ read: vi.fn(), save: vi.fn(), apply: vi.fn(), icon: vi.fn() }))
vi.mock('@/lib/api', () => ({ api: { desktop: { appearance: mocks.read, updateAppearance: mocks.save } } }))
vi.mock('@/lib/appearanceSync', () => ({ acceptAppearanceSnapshot: mocks.apply }))
vi.mock('@/lib/siteBranding', () => ({ prepareSiteIcon: mocks.icon }))
import SiteBrandingSettings from './SiteBrandingSettings.vue'

const current = { theme: 'dark', colors: null, wallpaper: 'prism', classicLevel: 'clear', resourceVersion: 'sha256:current', branding: { name: 'My server', icon: '' } }
beforeEach(() => {
  vi.resetAllMocks()
  mocks.read.mockResolvedValue(current)
  mocks.save.mockImplementation(async (body) => ({ ...body, configured: true, resourceVersion: 'sha256:saved' }))
})

describe('site branding settings', () => {
  it('saves trimmed branding with fresh theme values and applies only the confirmed response', async () => {
    const wrapper = mount(SiteBrandingSettings)
    await flushPromises()
    await wrapper.get('input[name="siteName"]').setValue('  New panel  ')
    mocks.read.mockResolvedValue({ ...current, wallpaper: 'orbit', resourceVersion: 'sha256:new' })
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(mocks.save).toHaveBeenCalledWith(expect.objectContaining({ branding: { name: 'New panel', icon: '' }, wallpaper: 'orbit', expectedResourceVersion: 'sha256:new' }))
    expect(mocks.apply).toHaveBeenCalledWith(expect.objectContaining({ branding: { name: 'New panel', icon: '' } }))
    expect(wrapper.get('[role="status"]').text()).toContain('已保存')
    wrapper.unmount()
  })

  it('preserves edits after a conflict and only applies defaults after saving', async () => {
    const wrapper = mount(SiteBrandingSettings)
    await flushPromises()
    await wrapper.get('input[name="siteName"]').setValue('Unsaved')
    mocks.save.mockRejectedValueOnce(new Error('conflict'))
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('输入已保留')
    expect((wrapper.get('input[name="siteName"]').element as HTMLInputElement).value).toBe('Unsaved')
    expect(mocks.apply).not.toHaveBeenCalled()
    await wrapper.get('button[type="button"]').trigger('click')
    expect(mocks.apply).not.toHaveBeenCalled()
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(mocks.save).toHaveBeenLastCalledWith(expect.objectContaining({ branding: { name: '', icon: '' } }))
    wrapper.unmount()
  })

  it('blocks invalid names and offers recovery after load failure', async () => {
    mocks.read.mockRejectedValueOnce(new Error('offline'))
    const wrapper = mount(SiteBrandingSettings)
    await flushPromises()
    expect(wrapper.find('form').exists()).toBe(false)
    await wrapper.get('button').trigger('click')
    await flushPromises()
    await wrapper.get('input[name="siteName"]').setValue('站'.repeat(65))
    await wrapper.get('form').trigger('submit')
    expect(mocks.save).not.toHaveBeenCalled()
    expect(wrapper.get('[role="alert"]').text()).toContain('64')
    wrapper.unmount()
  })
})
