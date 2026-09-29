// @vitest-environment jsdom
import { mount, flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import BackupScheduleDialog from './BackupScheduleDialog.vue'
import { backups, type BackupSettings } from '@/lib/backup'
vi.mock('@/lib/backup', async (original) => ({ ...await original<typeof import('@/lib/backup')>(), backups: { saveSchedule: vi.fn(), runSchedule: vi.fn() } }))
const settings: BackupSettings = { revision: 'r', storages: [], schedule: { enabled: true, modules: ['panel'], storageId: '', frequency: 'daily', hour: 3, minute: 0, weekday: 0, day: 1, timezone: 'Asia/Shanghai', keep: 7, hasPassword: true } }
function render() { return mount(BackupScheduleDialog, { props: { settings, pending: false }, global: { stubs: { ModalDialog: { template: '<div><slot /></div>' } } } }) }
describe('scheduled backups', () => {
  it('keeps the saved password on a blank edit and uses the selected wall time', async () => {
    vi.mocked(backups.saveSchedule).mockResolvedValue(settings)
    const wrapper = render()
    await wrapper.get('input[type=time]').setValue('04:30')
    await wrapper.get('form').trigger('submit'); await flushPromises()
    expect(backups.saveSchedule).toHaveBeenCalledWith('r', expect.objectContaining({ password: '', hour: 4, minute: 30, timezone: 'Asia/Shanghai' }))
    expect(wrapper.emitted('saved')).toHaveLength(1)
    wrapper.unmount()
  })
  it('requires matching new passwords without silently overwriting the saved one', async () => {
    vi.mocked(backups.saveSchedule).mockClear()
    const wrapper = render()
    await wrapper.get('input[type=password]').setValue('different-password')
    await wrapper.get('form').trigger('submit')
    expect(wrapper.get('button[type=submit]').attributes('disabled')).toBeDefined()
    expect(backups.saveSchedule).not.toHaveBeenCalled()
    wrapper.unmount()
  })
  it('runs the persisted plan and blocks duplicate runs while a backup is pending', async () => {
    vi.mocked(backups.runSchedule).mockResolvedValue({} as never)
    const wrapper = render()
    await wrapper.get('footer button[type=button]').trigger('click'); await flushPromises()
    expect(backups.runSchedule).toHaveBeenCalledOnce()
    expect(wrapper.emitted('started')).toHaveLength(1)
    await wrapper.setProps({ pending: true })
    expect(wrapper.get('footer button[type=button]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
})
