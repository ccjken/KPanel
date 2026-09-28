// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import ToastHost from './ToastHost.vue'
import { useToast } from '@/stores/toast'

const toast = useToast()
afterEach(() => { toast.items.value.forEach(item => toast.remove(item.id)); vi.useRealTimers() })
describe('actionable toast', () => {
  it('offers a native button, runs its action once, and removes the notice', async () => {
    vi.useFakeTimers()
    const wrapper = mount(ToastHost)
    const run = vi.fn()
    toast.show('新告警', { message: '<script>untrusted</script>', action: { label: '查看通知记录', run } })
    await nextTick()
    expect(wrapper.find('script').exists()).toBe(false)
    await wrapper.get('.toast__action').trigger('click')
    expect(run).toHaveBeenCalledTimes(1)
    expect(toast.items.value).toHaveLength(0)
    wrapper.unmount()
  })
})
