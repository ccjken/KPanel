// @vitest-environment jsdom
import { mount, flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useLiveNotifications } from './useLiveNotifications'
import { ApiError } from '@/lib/api'
import type { NotificationEvent } from '@/types/api'
import { resetDesktopModeForTest, useDesktopMode } from '@/stores/desktopMode'

const mocks = vi.hoisted(() => ({ history: vi.fn(), show: vi.fn(), remove: vi.fn(), push: vi.fn() }))
vi.mock('@/lib/api', () => ({ ApiError: class extends Error { constructor(message: string, public status: number) { super(message) } }, api: { cluster: { notificationHistory: mocks.history } } }))
vi.mock('@/stores/toast', () => ({ useToast: () => ({ show: mocks.show, remove: mocks.remove }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ push: mocks.push }) }))
const enabled = ref(true)
let visibility = 'visible'
let online = true
let wrapper: ReturnType<typeof mount> | undefined
const event = (id: string, kind: NotificationEvent['kind'] = 'alert') => ({ id, kind, hostName: '本机', message: 'CPU 96%', delivery: 'local_only' })
const page = (items: ReturnType<typeof event>[], nextCursor = '') => ({ items, nextCursor })
async function open() {
  wrapper = mount({ setup() { useLiveNotifications(enabled); return () => null } })
  await flushPromises()
}
async function tick() { await vi.advanceTimersByTimeAsync(15_000); await flushPromises() }
beforeEach(() => {
  vi.useFakeTimers(); vi.clearAllMocks()
  enabled.value = true; visibility = 'visible'; online = true
  resetDesktopModeForTest(); window.localStorage.clear()
  vi.spyOn(document, 'visibilityState', 'get').mockImplementation(() => visibility as DocumentVisibilityState)
  vi.spyOn(navigator, 'onLine', 'get').mockImplementation(() => online)
  mocks.show.mockReturnValue(42)
  mocks.history.mockResolvedValue(page([event('10')]))
})
afterEach(() => { wrapper?.unmount(); wrapper = undefined; vi.restoreAllMocks(); vi.useRealTimers() })

describe('live notifications', () => {
  it('silently baselines history, shows only new local events once, and links to the event', async () => {
    await open(); expect(mocks.show).not.toHaveBeenCalled()
    mocks.history.mockResolvedValue(page([event('11'), event('10')]))
    await tick()
    expect(mocks.show).toHaveBeenCalledTimes(1)
    expect(mocks.show).toHaveBeenCalledWith('新告警', expect.objectContaining({ tone: 'danger', message: '本机\nCPU 96%' }))
    mocks.show.mock.calls[0]![1].action.run()
    expect(mocks.push).toHaveBeenCalledWith({ path: '/activity', query: { tab: 'notifications', event: '11' } })
    await tick(); expect(mocks.show).toHaveBeenCalledTimes(1)
  })
  it('groups bursts into one notice and keeps IDs above JS safe integer precise', async () => {
    mocks.history.mockResolvedValue(page([event('9007199254740992')]))
    await open()
    mocks.history.mockResolvedValue(page([event('9007199254740994', 'recovery'), event('9007199254740993')]))
    await tick()
    expect(mocks.show).toHaveBeenCalledWith('2 条新通知', expect.objectContaining({ tone: 'danger' }))
    mocks.history.mockResolvedValue(page([event('9007199254740995', 'recovery')]))
    await tick()
    expect(mocks.remove).toHaveBeenCalledWith(42)
    expect(mocks.show).toHaveBeenLastCalledWith('恢复通知', expect.objectContaining({ tone: 'success' }))
  })
  it('pauses hidden/offline pages and groups unseen events when the page resumes', async () => {
    await open()
    visibility = 'hidden'; document.dispatchEvent(new Event('visibilitychange'))
    await tick(); expect(mocks.history).toHaveBeenCalledTimes(1)
    mocks.history.mockResolvedValue(page([event('12'), event('11')]))
    visibility = 'visible'; document.dispatchEvent(new Event('visibilitychange')); await flushPromises()
    expect(mocks.show).toHaveBeenCalledTimes(1)
    online = false; window.dispatchEvent(new Event('offline')); await tick()
    expect(mocks.history).toHaveBeenCalledTimes(2)
    online = true; window.dispatchEvent(new Event('online')); await flushPromises()
    expect(mocks.history).toHaveBeenCalledTimes(3)
    expect(mocks.show).toHaveBeenCalledTimes(1)
  })
  it('ignores late responses after logout, clears notices and baselines a new login', async () => {
    await open()
    mocks.history.mockResolvedValue(page([event('11')]))
    await tick()
    let resolve!: (value: ReturnType<typeof page>) => void
    mocks.history.mockImplementationOnce(() => new Promise(done => { resolve = done }))
    await tick()
    const signal = mocks.history.mock.calls.at(-1)![1] as AbortSignal
    enabled.value = false; await flushPromises()
    expect(signal.aborted).toBe(true); expect(mocks.remove).toHaveBeenCalledWith(42)
    resolve(page([event('12')])); await flushPromises(); await tick()
    expect(mocks.show).toHaveBeenCalledTimes(1)
    enabled.value = true; await flushPromises()
    expect(mocks.show).toHaveBeenCalledTimes(1)
  })
  it('does not poll unauthenticated/public routes and disposes polling on unmount', async () => {
    enabled.value = false; await open(); await tick()
    expect(mocks.history).not.toHaveBeenCalled()
    enabled.value = true; await flushPromises()
    expect(mocks.history).toHaveBeenCalledTimes(1)
    wrapper!.unmount(); wrapper = undefined
    await tick(); window.dispatchEvent(new Event('online')); await flushPromises()
    expect(mocks.history).toHaveBeenCalledTimes(1)
  })
  it('retries transient failures quietly and stops after session expiry', async () => {
    await open()
    mocks.history.mockRejectedValueOnce(new Error('offline'))
    await tick(); expect(mocks.show).not.toHaveBeenCalled()
    mocks.history.mockResolvedValueOnce(page([event('11')]))
    await tick(); expect(mocks.show).toHaveBeenCalledTimes(1)
    mocks.history.mockRejectedValueOnce(new ApiError('expired', 401))
    await tick(); const count = mocks.history.mock.calls.length
    await tick(); expect(mocks.history).toHaveBeenCalledTimes(count)
    expect(mocks.remove).toHaveBeenCalledWith(42)
  })
  it('rebaselines a restored store and detects its next new event', async () => {
    await open()
    mocks.history.mockResolvedValue(page([event('2')]))
    await tick(); expect(mocks.show).not.toHaveBeenCalled()
    mocks.history.mockResolvedValue(page([event('3', 'info')]))
    await tick(); expect(mocks.show).toHaveBeenCalledWith('新通知', expect.objectContaining({ tone: 'info' }))
  })
  it('bounds long host names and messages without splitting emoji', async () => {
    await open()
    mocks.history.mockResolvedValue(page([{ ...event('11'), hostName: '😀'.repeat(90), message: '😀'.repeat(170) }]))
    await tick()
    expect(mocks.show.mock.calls[0]![1].message).toBe('😀'.repeat(80) + '…\n' + '😀'.repeat(160) + '…')
  })
  it('aborts a stalled request after ten seconds then retries without overlap', async () => {
    let signal!: AbortSignal
    mocks.history.mockImplementationOnce((_filters, requestSignal: AbortSignal) => new Promise((_resolve, reject) => {
      signal = requestSignal
      signal.addEventListener('abort', () => reject(new DOMException('aborted', 'AbortError')))
    }))
    await open()
    await vi.advanceTimersByTimeAsync(10_000)
    expect(signal.aborted).toBe(true)
    expect(mocks.history).toHaveBeenCalledTimes(1)
    await tick()
    expect(mocks.history).toHaveBeenCalledTimes(2)
    expect(mocks.show).not.toHaveBeenCalled()
  })
  it('opens and reuses the activity desktop window instead of the global router', async () => {
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    await open()
    mocks.history.mockResolvedValue(page([event('11')]))
    await tick(); mocks.show.mock.calls.at(-1)![1].action.run()
    expect(desktop.windows.value).toHaveLength(1)
    const windowID = desktop.windows.value[0]!.id
    expect(desktop.windows.value[0]!.path).toBe('/activity?tab=notifications&event=11')
    desktop.minimizeWindow(windowID)
    mocks.history.mockResolvedValue(page([event('12')]))
    await tick(); mocks.show.mock.calls.at(-1)![1].action.run()
    expect(desktop.windows.value).toHaveLength(1)
    expect(desktop.windows.value[0]).toMatchObject({ id: windowID, path: '/activity?tab=notifications&event=12', minimized: false })
    expect(mocks.push).not.toHaveBeenCalled()
  })
  it('explains the desktop window limit without closing existing windows', async () => {
    const desktop = useDesktopMode()
    desktop.enterDesktop()
    for (let index = 0; index < 8; index++) desktop.openWindow('/files', 'route.files', true)
    await open()
    mocks.history.mockResolvedValue(page([event('11')]))
    await tick(); mocks.show.mock.calls.at(-1)![1].action.run()
    expect(desktop.windows.value).toHaveLength(8)
    expect(mocks.show.mock.calls.at(-1)![1].message).toBeTruthy()
    expect(mocks.push).not.toHaveBeenCalled()
  })
})
