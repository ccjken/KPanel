import { readonly, ref } from 'vue'
import { applySiteBranding } from '@/stores/branding'
import { ApiError, api } from '@/lib/api'
import { useClassicWallpaper } from '@/lib/classicWallpaper'
import { applySyncedWallpaper, isDesktopWallpaperID, useDesktopWallpaper } from '@/lib/desktopWallpapers'
import { useTheme } from '@/stores/theme'
import { useToast } from '@/stores/toast'
import { parseStoredThemeColors } from '@/theme/colors'
import { t } from '@/i18n'
import type { AppearanceSettings } from '@/types/api'

type AppearanceValue = Pick<AppearanceSettings, 'theme' | 'colors' | 'wallpaper' | 'classicLevel' | 'branding'>
type Patch = Partial<AppearanceValue>
// Per-tab explicit edits survive refresh; a cached full snapshot never becomes a write.
const PENDING_KEY = 'kpanel:appearance-pending:v1'
const theme = useTheme()
const wallpaper = useDesktopWallpaper()
const classic = useClassicWallpaper()
const ready = ref(false)
export const appearanceReady = readonly(ready)

interface Sync {
  owner: string
  server?: AppearanceSettings
  pending: Patch
  initial: AppearanceValue
  task?: Promise<void>
  timer?: number
  retries: number
  notified: boolean
}
let current: Sync | undefined
let applying = false
let operationTail: Promise<void> = Promise.resolve()

function serializeAppearance<T>(operation: () => Promise<T>): Promise<T> {
  const task = operationTail.then(operation)
  operationTail = task.then(() => {}, () => {})
  return task
}

function localValue(): AppearanceValue {
  return {
    theme: theme.preference.value,
    colors: theme.isCustom.value ? { ...theme.colors.value } : null,
    wallpaper: wallpaper.id.value,
    classicLevel: classic.level.value,
  }
}

function readPending(owner: string): Patch {
  try {
    const raw = sessionStorage.getItem(PENDING_KEY)
    const stored = raw && raw.length < 4096 ? JSON.parse(raw) : null
    if (stored?.owner !== owner || !stored.patch || typeof stored.patch !== 'object') return {}
    const value = stored.patch
    const patch: Patch = {}
    if (['system', 'light', 'dark'].includes(value.theme)) patch.theme = value.theme
    if (typeof value.wallpaper === 'string' && isDesktopWallpaperID(value.wallpaper)) patch.wallpaper = value.wallpaper
    if (['off', 'ambient', 'clear'].includes(value.classicLevel)) patch.classicLevel = value.classicLevel
    if (value.colors === null) patch.colors = null
    else if (value.colors) {
      const colors = parseStoredThemeColors(JSON.stringify({ ...value.colors, version: 1 }))
      if (colors) patch.colors = colors
    }
    return patch
  } catch { return {} }
}

function persist(sync: Sync): void {
  try {
    if (Object.keys(sync.pending).length) sessionStorage.setItem(PENDING_KEY, JSON.stringify({ owner: sync.owner, patch: sync.pending }))
    else sessionStorage.removeItem(PENDING_KEY)
  } catch { /* Memory still preserves edits while storage is unavailable. */ }
}

function apply(value: AppearanceValue): void {
  applySiteBranding(value.branding)
  applying = true
  try {
    // Keep the existing stores as the UI facade, with one reconciliation path.
    if (theme.preference.value !== value.theme) theme.setTheme(value.theme)
    if (JSON.stringify(theme.isCustom.value ? theme.colors.value : null) !== JSON.stringify(value.colors)) {
      if (value.colors) theme.setColors(value.colors)
      else theme.resetColors()
    }
    if (isDesktopWallpaperID(value.wallpaper)) applySyncedWallpaper(value.wallpaper)
    classic.setLevel(value.classicLevel)
  } finally { applying = false }
}

function changed(event: Event): void {
  const sync = current
  if (!sync || applying) return
  Object.assign(sync.pending, (event as CustomEvent<Patch>).detail)
  persist(sync)
  sync.retries = 0
  // A wallpaper and its matching colors are emitted in the same turn.
  queueMicrotask(() => { if (current === sync) void reconcile(sync) })
}

function retry(): void {
  if (!current) return
  current.retries = 0
  void reconcile(current)
}

function reconcile(sync: Sync): Promise<void> {
  if (current !== sync) return Promise.resolve()
  if (sync.task) return sync.task
  window.clearTimeout(sync.timer)
  sync.timer = undefined
  sync.task = serializeAppearance(async () => {
    let conflicts = 0
    try {
      if (current !== sync) return
      if (!sync.server) {
        const fetched = await api.desktop.appearance()
        if (current !== sync) return
        sync.server = fetched
      }
      const initial = sync.initial
      let migration: Patch = !sync.server.configured
        && (initial.theme !== 'system' || initial.colors !== null || initial.wallpaper !== 'classic' || initial.classicLevel !== 'off') ? initial : {}
      // Implicit legacy migration must never survive a competing server configuration.
      // Only explicit user edits are durable or replayed after a conflict.
      apply({ ...sync.server, ...migration, ...sync.pending })
      ready.value = true
      while (current === sync && (Object.keys(migration).length || Object.keys(sync.pending).length)) {
        const patch = { ...migration, ...sync.pending }
        const next = { ...sync.server, ...patch }
        try {
          const updated = await api.desktop.updateAppearance({
            theme: next.theme, colors: next.colors, wallpaper: next.wallpaper,
            classicLevel: next.classicLevel, expectedResourceVersion: sync.server.resourceVersion,
          })
          if (current !== sync) return
          sync.server = updated
          migration = {}
          // An edit made during this request is kept for the next serialized save.
          for (const key of Object.keys(patch) as (keyof Patch)[]) {
            if (JSON.stringify(sync.pending[key]) === JSON.stringify(patch[key])) delete sync.pending[key]
          }
          persist(sync)
          apply({ ...sync.server, ...sync.pending })
        } catch (error) {
          if (current !== sync) return
          if (!(error instanceof ApiError && error.status === 409) || ++conflicts > 3) throw error
          const fetched = await api.desktop.appearance()
          if (current !== sync) return
          sync.server = fetched
          if (sync.server.configured) migration = {}
          apply({ ...sync.server, ...sync.pending })
        }
      }
      sync.retries = 0
      sync.notified = false
    } catch {
      if (current !== sync) return
      if (!sync.notified) {
        useToast().danger(t(sync.server ? 'desktop.appearanceSyncFailed' : 'desktop.appearanceLoadFailed'), t('desktop.appearanceSyncRetry'))
        sync.notified = true
      }
      // Bounded backoff, then explicit edits, reconnect or refresh can try again.
      if (sync.retries < 4) sync.timer = window.setTimeout(() => { void reconcile(sync) }, 1000 * 2 ** sync.retries++)
    } finally { sync.task = undefined }
  })
  return sync.task
}

/** Explicit saves share the theme sync queue, including response application. */
export function saveAppearance(operation: () => Promise<AppearanceSettings>, signal: AbortSignal): Promise<AppearanceSettings> {
  const owner = current
  return serializeAppearance(async () => {
    signal.throwIfAborted()
    if (current !== owner) throw new Error('Appearance session changed')
    const snapshot = await operation()
    if (current === owner && !signal.aborted) {
      if (owner) owner.server = snapshot
      apply({ ...snapshot, ...owner?.pending })
    }
    return snapshot
  })
}

/** Session/login responses seed this store; older servers use the same GET fallback. */
export function startAppearanceSync(snapshot?: AppearanceSettings, owner = ''): Promise<void> {
  if (current) return current.task ?? Promise.resolve()
  const sync: Sync = { owner, server: snapshot, pending: readPending(owner), initial: localValue(), retries: 0, notified: false }
  current = sync
  ready.value = false
  window.addEventListener('kpanel:appearance-changed', changed)
  window.addEventListener('online', retry)
  return reconcile(sync)
}

export function stopAppearanceSync(): void {
  if (current) window.clearTimeout(current.timer)
  current = undefined
  // A stopped session's pending network request must not hold the next session.
  // Each queued callback still checks its captured owner before doing any work.
  operationTail = Promise.resolve()
  ready.value = false
  window.removeEventListener('kpanel:appearance-changed', changed)
  window.removeEventListener('online', retry)
}
