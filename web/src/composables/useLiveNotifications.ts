import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import { useRouter } from 'vue-router'
import { ApiError, api } from '@/lib/api'
import { t } from '@/i18n'
import { useToast } from '@/stores/toast'
import { useDesktopMode } from '@/stores/desktopMode'

const pollInterval = 15_000
function excerpt(value: string, limit: number): string {
  const characters = Array.from(value)
  return characters.length > limit ? characters.slice(0, limit).join('') + '…' : value
}

/** One subscription per authenticated app, across classic and desktop navigation. */
export function useLiveNotifications(enabled: Readonly<Ref<boolean>>): void {
  const router = useRouter()
  const toast = useToast()
  const desktop = useDesktopMode()
  let mounted = false
  let stopped = false
  let generation = 0
  let watermark: bigint | undefined
  let timer: number | undefined
  let pending: AbortController | undefined
  let noticeID: number | undefined

  function clearNotice(): void {
    if (noticeID !== undefined) toast.remove(noticeID)
    noticeID = undefined
  }

  function cancel(): void {
    generation++
    window.clearTimeout(timer)
    pending?.abort()
    pending = undefined
  }

  function available(): boolean {
    return mounted && enabled.value && !stopped && document.visibilityState === 'visible' && navigator.onLine
  }

  async function poll(): Promise<void> {
    if (!available() || pending) return
    window.clearTimeout(timer)
    const controller = new AbortController()
    pending = controller
    const run = generation
    const timeout = window.setTimeout(() => controller.abort(), 10_000)
    try {
      const page = await api.cluster.notificationHistory({ limit: '50' }, controller.signal)
      if (run !== generation || !available() || controller.signal.aborted) return
      const events = page.items.filter(event => /^[1-9]\d{0,19}$/.test(event.id))
      const latest = events.reduce((max, event) => BigInt(event.id) > max ? BigInt(event.id) : max, 0n)
      // First load establishes a baseline; a restored/cleared store establishes a new one.
      if (watermark === undefined || latest < watermark) { watermark = latest; return }
      const fresh = events.filter(event => BigInt(event.id) > watermark!)
      if (!fresh.length) return
      watermark = latest
      const newest = fresh.reduce((a, b) => BigInt(a.id) > BigInt(b.id) ? a : b)
      const count = page.nextCursor && fresh.length === 50 ? '50+' : fresh.length
      const title = fresh.length > 1 ? t('notifications.newEvents', { count })
        : t(newest.kind === 'alert' ? 'notifications.alert' : newest.kind === 'recovery' ? 'notifications.recovery' : 'notifications.event')
      clearNotice()
      noticeID = toast.show(title, {
        message: `${excerpt(newest.hostName, 80)}\n${excerpt(newest.message, 160)}`,
        tone: fresh.some(event => event.kind === 'alert') ? 'danger' : fresh.every(event => event.kind === 'recovery') ? 'success' : 'info',
        duration: 12_000,
        action: {
          label: t('notifications.viewHistory'),
          run: () => {
            if (desktop.mode.value === 'desktop') {
              const windowID = desktop.openWindow(`/activity?tab=notifications&event=${newest.id}`, 'route.activity', false, true)
              if (!windowID) toast.show(t('desktop.windowLimitTitle'), { message: t('desktop.windowLimitMessage') })
            } else {
              void router.push({ path: '/activity', query: { tab: 'notifications', event: newest.id } })
            }
          },
        },
      })
    } catch (error) {
      if (run !== generation) return
      // Optional live hints must not flood the page with transport errors.
      if (error instanceof ApiError && [401, 403, 404].includes(error.status)) {
        stopped = true
        clearNotice()
      }
    } finally {
      window.clearTimeout(timeout)
      if (run === generation) {
        pending = undefined
        if (available()) timer = window.setTimeout(() => void poll(), pollInterval)
      }
    }
  }

  function resume(): void {
    cancel()
    if (available()) void poll()
    else clearNotice()
  }

  watch(enabled, () => {
    cancel()
    watermark = undefined
    stopped = false
    clearNotice()
    if (available()) void poll()
  })
  onMounted(() => {
    mounted = true
    document.addEventListener('visibilitychange', resume)
    window.addEventListener('online', resume)
    window.addEventListener('offline', resume)
    void poll()
  })
  onBeforeUnmount(() => {
    mounted = false
    cancel()
    clearNotice()
    document.removeEventListener('visibilitychange', resume)
    window.removeEventListener('online', resume)
    window.removeEventListener('offline', resume)
  })
}
