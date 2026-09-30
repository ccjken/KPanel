// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  appearance: vi.fn(),
  updateAppearance: vi.fn(),
  setTheme: vi.fn(),
  setColors: vi.fn(),
  resetColors: vi.fn(),
  setLevel: vi.fn(),
  applyWallpaper: vi.fn(),
  danger: vi.fn(),
  preference: { value: 'system' },
  colors: { value: { brand: '#2867b2', neutral: '#4b5d76', signature: '#2867b2', signatureLinked: true } },
  isCustom: { value: false },
  wallpaper: { value: 'classic' },
  classicLevel: { value: 'off' },
}))

vi.mock('@/lib/api', () => ({ api: { desktop: { appearance: mocks.appearance, updateAppearance: mocks.updateAppearance } }, ApiError: class extends Error { constructor(message: string, public status = 0) { super(message) } } }))
vi.mock('@/stores/theme', () => ({ useTheme: () => ({ preference: mocks.preference, colors: mocks.colors, isCustom: mocks.isCustom, setTheme: mocks.setTheme, setColors: mocks.setColors, resetColors: mocks.resetColors }) }))
vi.mock('@/lib/classicWallpaper', () => ({ useClassicWallpaper: () => ({ level: mocks.classicLevel, setLevel: mocks.setLevel }) }))
vi.mock('@/lib/desktopWallpapers', () => ({ useDesktopWallpaper: () => ({ id: mocks.wallpaper }), isDesktopWallpaperID: () => true, applySyncedWallpaper: mocks.applyWallpaper }))
vi.mock('@/stores/toast', () => ({ useToast: () => ({ danger: mocks.danger }) }))
vi.mock('@/i18n', () => ({ t: (key: string) => key }))

import { appearanceReady, startAppearanceSync, stopAppearanceSync } from './appearanceSync'

const remote = {
  configured: true, resourceVersion: 'sha256:remote', theme: 'dark' as const, colors: null,
  wallpaper: 'orbit', classicLevel: 'clear' as const,
}

async function settle(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 0))
}

describe('shared appearance preference', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    sessionStorage.clear()
    mocks.preference.value = 'system'
    mocks.isCustom.value = false
    mocks.wallpaper.value = 'classic'
    mocks.classicLevel.value = 'off'
  })
  afterEach(() => { stopAppearanceSync(); vi.useRealTimers() })

  it('holds the first wallpaper until a fresh browser receives the server choice', async () => {
    let resolve!: (value: typeof remote) => void
    mocks.appearance.mockReturnValue(new Promise((done) => { resolve = done }))
    const sync = startAppearanceSync()
    expect(appearanceReady.value).toBe(false)
    expect(mocks.applyWallpaper).not.toHaveBeenCalled()
    resolve(remote)
    await sync
    expect(mocks.applyWallpaper).toHaveBeenCalledWith('orbit')
    expect(appearanceReady.value).toBe(true)
  })

  it('keeps a neutral surface on failure rather than guessing a default wallpaper', async () => {
    mocks.appearance.mockRejectedValue(new Error('offline'))
    await startAppearanceSync()
    expect(appearanceReady.value).toBe(false)
    expect(mocks.applyWallpaper).not.toHaveBeenCalled()
    expect(mocks.danger).toHaveBeenCalledWith('desktop.appearanceLoadFailed', 'desktop.appearanceSyncRetry')
  })

  it('does not reveal a new session when an old appearance request finishes after logout', async () => {
    let resolveOld!: (value: typeof remote) => void
    let resolveNew!: (value: typeof remote) => void
    mocks.appearance.mockReturnValueOnce(new Promise((done) => { resolveOld = done }))
      .mockReturnValueOnce(new Promise((done) => { resolveNew = done }))
    const old = startAppearanceSync()
    await Promise.resolve()
    stopAppearanceSync()
    const current = startAppearanceSync()
    resolveOld(remote)
    await old
    expect(appearanceReady.value).toBe(false)
    expect(mocks.applyWallpaper).not.toHaveBeenCalled()
    resolveNew(remote)
    await current
    expect(appearanceReady.value).toBe(true)
  })

  it('applies the saved server choice in a fresh browser and writes later changes', async () => {
    mocks.appearance.mockResolvedValue({ ...remote })
    mocks.updateAppearance.mockImplementation(async (body) => ({ ...remote, ...body, configured: true, resourceVersion: 'sha256:next' }))
    await startAppearanceSync()
    expect(mocks.setTheme).toHaveBeenCalledWith('dark')
    expect(mocks.applyWallpaper).toHaveBeenCalledWith('orbit')
    expect(mocks.setLevel).toHaveBeenCalledWith('clear')
    expect(mocks.updateAppearance).not.toHaveBeenCalled()

    window.dispatchEvent(new CustomEvent('kpanel:appearance-changed', { detail: { wallpaper: 'prism' } }))
    await settle()
    expect(mocks.updateAppearance).toHaveBeenCalledWith(expect.objectContaining({
      theme: 'dark', wallpaper: 'prism', classicLevel: 'clear', expectedResourceVersion: 'sha256:remote',
    }))
  })

  it('does not let a new browser with defaults replace an unconfigured server', async () => {
    mocks.appearance.mockResolvedValue({ ...remote, configured: false, theme: 'system', wallpaper: 'classic', classicLevel: 'off' })
    mocks.updateAppearance.mockImplementation(async (body) => ({ ...remote, ...body, resourceVersion: 'sha256:next' }))
    await startAppearanceSync()
    expect(mocks.updateAppearance).not.toHaveBeenCalled()
    window.dispatchEvent(new CustomEvent('kpanel:appearance-changed', { detail: { theme: 'light' } }))
    await settle()
    expect(mocks.updateAppearance).toHaveBeenCalledWith(expect.objectContaining({ theme: 'light', expectedResourceVersion: 'sha256:remote' }))
  })

  it('rebases an explicit change when another browser saved first', async () => {
    const { ApiError } = await import('@/lib/api')
    mocks.appearance.mockResolvedValueOnce({ ...remote }).mockResolvedValueOnce({ ...remote, theme: 'light', resourceVersion: 'sha256:other' })
    mocks.updateAppearance.mockRejectedValueOnce(new ApiError('conflict', 409))
      .mockImplementationOnce(async (body) => ({ ...remote, ...body, configured: true, resourceVersion: 'sha256:next' }))
    await startAppearanceSync()
    window.dispatchEvent(new CustomEvent('kpanel:appearance-changed', { detail: { wallpaper: 'prism' } }))
    await settle()
    expect(mocks.setTheme).toHaveBeenLastCalledWith('light')
    expect(mocks.updateAppearance).toHaveBeenNthCalledWith(2, expect.objectContaining({
      theme: 'light', wallpaper: 'prism', expectedResourceVersion: 'sha256:other',
    }))
  })
})

describe('appearance recovery and request budget', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    sessionStorage.clear()
    mocks.preference.value = 'system'
    mocks.isCustom.value = false
    mocks.wallpaper.value = 'classic'
    mocks.classicLevel.value = 'off'
    mocks.updateAppearance.mockImplementation(async (body) => ({ ...remote, ...body, configured: true, resourceVersion: 'sha256:next' }))
  })
  afterEach(() => { stopAppearanceSync(); vi.useRealTimers() })

  const change = (detail: Record<string, unknown>) => window.dispatchEvent(new CustomEvent('kpanel:appearance-changed', { detail }))

  it('reuses the authenticated snapshot and deduplicates multiple starts', async () => {
    await Promise.all([startAppearanceSync(remote, 'admin'), startAppearanceSync(remote, 'admin')])
    expect(mocks.appearance).not.toHaveBeenCalled()
    change({ theme: 'light' })
    change({ wallpaper: 'prism' })
    await settle()
    expect(mocks.updateAppearance).toHaveBeenCalledTimes(1)
    expect(mocks.updateAppearance).toHaveBeenCalledWith(expect.objectContaining({ theme: 'light', wallpaper: 'prism' }))
  })

  it('retries a failed initial read and saves edits made while disconnected', async () => {
    vi.useFakeTimers()
    mocks.appearance.mockRejectedValueOnce(new Error('offline')).mockResolvedValue(remote)
    await startAppearanceSync()
    expect(appearanceReady.value).toBe(false)
    change({ theme: 'light' })
    await vi.advanceTimersByTimeAsync(1000)
    expect(appearanceReady.value).toBe(true)
    expect(mocks.updateAppearance).toHaveBeenCalledWith(expect.objectContaining({ theme: 'light', wallpaper: 'orbit' }))
  })

  it('bounds automatic retries and resumes after reconnect', async () => {
    vi.useFakeTimers()
    mocks.appearance.mockRejectedValue(new Error('offline'))
    await startAppearanceSync()
    await vi.advanceTimersByTimeAsync(60_000)
    expect(mocks.appearance).toHaveBeenCalledTimes(5)
    expect(mocks.danger).toHaveBeenCalledTimes(1)
    mocks.appearance.mockResolvedValue(remote)
    window.dispatchEvent(new Event('online'))
    await vi.advanceTimersByTimeAsync(0)
    expect(appearanceReady.value).toBe(true)
  })

  it('replays only explicit unsaved fields after refresh, retaining another device changes', async () => {
    mocks.updateAppearance.mockRejectedValueOnce(new Error('offline'))
    await startAppearanceSync(remote, 'admin')
    change({ wallpaper: 'prism' })
    await settle()
    expect(sessionStorage.getItem('kpanel:appearance-pending:v1')).toContain('prism')
    stopAppearanceSync()
    await startAppearanceSync({ ...remote, theme: 'light', resourceVersion: 'sha256:other' }, 'admin')
    expect(mocks.updateAppearance).toHaveBeenLastCalledWith(expect.objectContaining({ theme: 'light', wallpaper: 'prism', expectedResourceVersion: 'sha256:other' }))
    expect(mocks.setTheme).toHaveBeenLastCalledWith('light')
    expect(sessionStorage.getItem('kpanel:appearance-pending:v1')).toBeNull()
  })

  it('does not replay another account edits', async () => {
    sessionStorage.setItem('kpanel:appearance-pending:v1', JSON.stringify({ owner: 'old-user', patch: { theme: 'light' } }))
    await startAppearanceSync(remote, 'new-user')
    expect(mocks.updateAppearance).not.toHaveBeenCalled()
    expect(mocks.setTheme).toHaveBeenLastCalledWith('dark')
  })

  it('keeps edits made during an in-flight save and persists the in-flight patch', async () => {
    let resolve!: (value: typeof remote) => void
    mocks.updateAppearance.mockReturnValueOnce(new Promise(done => { resolve = done }))
    await startAppearanceSync(remote)
    change({ theme: 'light' })
    await settle()
    expect(sessionStorage.getItem('kpanel:appearance-pending:v1')).toContain('light')
    change({ theme: 'dark', wallpaper: 'prism' })
    await settle()
    expect(mocks.updateAppearance).toHaveBeenCalledTimes(1)
    resolve({ ...remote, theme: 'light' as 'dark' })
    await settle()
    expect(mocks.updateAppearance).toHaveBeenCalledTimes(2)
    expect(mocks.updateAppearance).toHaveBeenLastCalledWith(expect.objectContaining({ theme: 'dark', wallpaper: 'prism' }))
    expect(mocks.applyWallpaper).toHaveBeenLastCalledWith('prism')
  })

  it('ignores old save completion after stop and does not unlock a new session writer', async () => {
    let finishOld!: (value: typeof remote) => void
    let finishNew!: (value: typeof remote) => void
    mocks.updateAppearance
      .mockReturnValueOnce(new Promise(done => { finishOld = done }))
      .mockReturnValueOnce(new Promise(done => { finishNew = done }))
    await startAppearanceSync(remote, 'first')
    change({ theme: 'light' }); await settle()
    stopAppearanceSync()
    await startAppearanceSync(remote, 'second')
    change({ wallpaper: 'prism' }); await settle()
    finishOld(remote); await settle()
    change({ wallpaper: 'rift' }); await settle()
    expect(mocks.updateAppearance).toHaveBeenCalledTimes(2)
    finishNew(remote); await settle()
    expect(mocks.updateAppearance).toHaveBeenCalledTimes(3)
    expect(mocks.updateAppearance).toHaveBeenLastCalledWith(expect.objectContaining({ wallpaper: 'rift' }))
  })

  it('cancels retry timers when leaving the session', async () => {
    vi.useFakeTimers()
    mocks.appearance.mockRejectedValue(new Error('offline'))
    await startAppearanceSync()
    stopAppearanceSync()
    await vi.advanceTimersByTimeAsync(60_000)
    expect(mocks.appearance).toHaveBeenCalledTimes(1)
  })
})
