import type { PublicClusterShareSnapshot } from '@/types/api'
import type { ScenePackList, LocalizedText } from './scenePacks'
import { clusterTrafficCounters, formatNetworkTrafficCounter, monthlyTrafficUsage, clusterTrafficHint } from './networkTraffic'
import type { useI18n } from '@/i18n'
import { formatClusterMoney, estimateRemainingValue, summarizeRemainingValue } from './clusterRemainingValue'
import { formatPercent, formatDuration } from './format'

export interface ShareTheme { id: string; name: LocalizedText; fileBase: string }
export interface ShareThemeList extends ScenePackList { selected?: string; resourceVersion: string }

export function shareThemeURL(theme?: ShareTheme): string | undefined {
  if (!theme || !/^[a-z0-9][a-z0-9-]{0,39}$/.test(theme.id)) return
  const prefix = `/api/v1/cluster/share-themes/${theme.id}/files/`
  if (!theme.fileBase.startsWith(prefix) || !/^[a-f0-9]{32}\/$/.test(theme.fileBase.slice(prefix.length))) return
  return `${theme.fileBase}index.html`
}

// Explicit display DTO: never forward a session, share URL/token, arbitrary API
// additions, or management objects to a community package.
export function shareThemeModel(snapshot: PublicClusterShareSnapshot, locale: string, now = new Date(), t?: ReturnType<typeof useI18n>['t']) {
  const details = Object.fromEntries(snapshot.items.map(host => [host.id, host]))
  const summary = summarizeRemainingValue(snapshot.items, details, now)
  return {
    title: snapshot.title, description: snapshot.description || '', generatedAt: snapshot.generatedAt,
    total: snapshot.total, online: snapshot.online, attention: snapshot.attention,
    value: { included: summary.included, excluded: summary.excluded,
      groups: summary.groups.map(group => ({ currency: group.currency, text: formatClusterMoney(group.remaining, group.currency, locale) })) },
    hosts: snapshot.items.map(host => {
      const usage = monthlyTrafficUsage(host.trafficPeriod, host)
      const counters = clusterTrafficCounters(host)
      const estimate = estimateRemainingValue(host, now)
      return {
        id: host.id, name: host.name, state: host.state, os: host.os || '',
        location: [host.location.country, host.location.city].filter(Boolean).join(' · '),
        cpu: host.collectedAt ? formatPercent(host.cpu.usagePercent) : '—',
        memory: host.collectedAt ? formatPercent(host.memory.usagePercent) : '—',
        disk: host.collectedAt ? formatPercent(host.disk.usagePercent) : '—',
        uptime: host.collectedAt ? formatDuration(host.uptimeSeconds || 0) : '—',
        traffic: { monthly: Boolean(host.trafficPeriod || host.trafficResetDay), percent: usage?.text || '', tone: usage?.tone || 'normal',
          hint: t ? clusterTrafficHint(host.trafficPeriod, host, t) || '' : '',
          available: host.trafficPeriod?.available ?? Boolean(host.collectedAt), partial: host.trafficPeriod?.partial || false,
          estimated: host.trafficPeriod?.estimated || false,
          received: host.collectedAt ? formatNetworkTrafficCounter(counters, 'received') : '—',
          sent: host.collectedAt ? formatNetworkTrafficCounter(counters, 'sent') : '—' },
        price: host.price || '', expiresOn: host.expiresOn || '',
        remaining: estimate.remaining === undefined ? '—' : formatClusterMoney(estimate.remaining, estimate.currency, locale),
      }
    }),
  }
}
