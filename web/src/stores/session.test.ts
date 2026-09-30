// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AuthStatus } from '@/types/api'

const mocks = vi.hoisted(() => ({
  status: vi.fn(), login: vi.fn(), logout: vi.fn(), reset: vi.fn(), stop: vi.fn(), apply: vi.fn(), cancel: vi.fn(),
}))
vi.mock('@/lib/api', () => ({ api: { auth: { status: mocks.status, login: mocks.login, logout: mocks.logout } }, resetApiSecurityState: mocks.reset }))
vi.mock('@/lib/appearanceSync', () => ({ stopAppearanceSync: mocks.stop }))
vi.mock('@/lib/loginAppearance', () => ({ applyLoginAppearance: mocks.apply, cancelLoginWallpaper: mocks.cancel }))
import { useSession } from './session'

const appearance = { configured: true, resourceVersion: 'sha256:current', theme: 'dark' as const, colors: null, wallpaper: 'prism', classicLevel: 'off' as const }
const guest: AuthStatus = {
  setupRequired: false, authenticated: false,
  loginAppearance: { theme: 'dark', colors: null, wallpaper: { url: '/wallpapers/kpanel-desktop-prism.webp', focusX: 500, focusY: 500, bright: false } },
}
const authenticated: AuthStatus = { setupRequired: false, authenticated: true, user: { id: 'admin', username: 'admin' }, appearance }

describe('session appearance handoff', () => {
  beforeEach(() => vi.resetAllMocks())
  it('carries authenticated appearance to the shell and reloads guest branding on logout', async () => {
    mocks.login.mockResolvedValue(authenticated)
    mocks.logout.mockResolvedValue(undefined)
    mocks.status.mockResolvedValue(guest)
    const session = useSession()
    await session.login({ username: 'admin', password: 'fixture-only' })
    expect(session.state.appearance).toEqual(appearance)
    await session.logout()
    expect(mocks.stop).toHaveBeenCalledTimes(1)
    expect(mocks.status).toHaveBeenCalledTimes(1)
    expect(mocks.apply).toHaveBeenCalledWith(guest.loginAppearance)
    expect(session.state.authenticated).toBe(false)
    expect(session.state.appearance).toBeUndefined()
  })

  it('does not restore an old authenticated state after logout refresh', async () => {
    let resolve!: (value: AuthStatus) => void
    mocks.status.mockReturnValueOnce(new Promise(done => { resolve = done })).mockResolvedValueOnce(guest)
    mocks.logout.mockResolvedValue(undefined)
    const session = useSession()
    const old = session.refresh(true)
    await session.logout()
    resolve(authenticated)
    await old
    expect(session.state.authenticated).toBe(false)
    expect(mocks.apply).toHaveBeenLastCalledWith(guest.loginAppearance)
  })

  it('deduplicates session reads and prevents a slow guest response from replacing a login', async () => {
    let resolve!: (value: AuthStatus) => void
    mocks.status.mockReturnValueOnce(new Promise(done => { resolve = done }))
    mocks.login.mockResolvedValue(authenticated)
    const session = useSession()
    const old = session.refresh(true)
    const same = session.refresh()
    expect(mocks.status).toHaveBeenCalledTimes(1)
    await session.login({ username: 'admin', password: 'fixture-only' })
    resolve(guest)
    await Promise.all([old, same])
    expect(session.state.authenticated).toBe(true)
    expect(session.state.appearance).toEqual(appearance)
    expect(mocks.apply).not.toHaveBeenCalled()
  })
})
