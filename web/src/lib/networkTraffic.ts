import { formatBytes } from '@/lib/format'
import type { ClusterHostDetails, ClusterTrafficPeriod } from '@/types/api'
import type { useI18n } from '@/i18n'

export function monthlyTrafficUsage(period?: ClusterTrafficPeriod, details?: ClusterHostDetails) {
  const quota = details?.trafficMonthlyQuotaGiB
  if (!period?.available || !quota || !Number.isInteger(quota) || quota < 1 || quota > 1_048_576) return undefined
  const received = finiteCounter(period.receivedBytes)
  const sent = finiteCounter(period.sentBytes)
  if (received === undefined || sent === undefined) return undefined
  const calculation = details?.trafficCalculation || 'total'
  let used: number
  switch (calculation) {
    case 'received': used = received; break
    case 'sent': used = sent; break
    case 'max': used = Math.max(received, sent); break
    case 'total': used = received + sent; break
    default: return undefined
  }
  const percent = used * 100 / (quota * 1024 ** 3)
  // Floor avoids displaying 80/95/100% before the corresponding boundary is reached.
  return { percent, text: `${Math.floor(percent)}%`, used, quotaBytes: quota * 1024 ** 3, calculation,
    tone: percent >= 95 ? 'danger' : percent >= 80 ? 'warning' : 'normal' }
}

export function clusterTrafficHint(period: ClusterTrafficPeriod | undefined, details: ClusterHostDetails | undefined, t: ReturnType<typeof useI18n>['t']) {
  const parts = [trafficPeriodHint(period, t)]
  if (!period && details?.trafficResetDay) parts.push(t('cluster.traffic.waiting'))
  const usage = monthlyTrafficUsage(period, details)
  if (usage) {
    parts.push(t('cluster.traffic.quotaSummary', { used: formatBytes(usage.used), quota: formatBytes(usage.quotaBytes),
      method: t(`cluster.traffic.calculation.${usage.calculation}`) }))
    if (usage.percent >= 100) parts.push(t('cluster.traffic.exceeded'))
    else if (usage.percent >= 80) parts.push(t('cluster.traffic.nearQuota'))
  }
  return parts.filter(Boolean).join(' · ') || undefined
}

export function clusterTrafficCounters(host: {
  trafficPeriod?: ClusterTrafficPeriod
  lastSnapshot?: { telemetry: { network: NetworkTrafficCounters } }
  network?: NetworkTrafficCounters
} | undefined): NetworkTrafficCounters | undefined {
  if (host?.trafficPeriod) return host.trafficPeriod.available ? host.trafficPeriod : undefined
  return host?.lastSnapshot?.telemetry.network ?? host?.network
}

export function trafficPeriodHint(period: ClusterTrafficPeriod | undefined, t: ReturnType<typeof useI18n>['t']): string | undefined {
  if (!period) return undefined
  if (!period.available) return t('cluster.traffic.waiting')
  const parts = [t('cluster.traffic.period', { start: period.startedAt, end: period.endsAt })]
  if (period.partial) parts.push(t('cluster.traffic.partial'))
  if (period.estimated) parts.push(t('cluster.traffic.estimated'))
  return parts.join(' · ')
}

/**
 * The panel receives the same monotonic byte counters through two API shapes:
 * the host summary uses receivedBytes/sentBytes, while the desktop monitor's
 * normalized snapshot calls them totalReceivedBytes/totalTransmittedBytes.
 * Keep the mapping and directional formatting in one place so every view uses
 * the same binary units and rounding without inventing an aggregate metric.
 */
export interface NetworkTrafficCounters {
  receivedBytes?: number
  sentBytes?: number
  totalReceivedBytes?: number
  totalTransmittedBytes?: number
}

export type NetworkTrafficDirection = 'received' | 'sent'

function finiteCounter(value?: number): number | undefined {
  if (value === undefined || !Number.isFinite(value) || value < 0) return undefined
  return value
}

export function networkTrafficCounterBytes(
  value: NetworkTrafficCounters | undefined,
  direction: NetworkTrafficDirection,
): number | undefined {
  if (!value) return undefined
  if (direction === 'received') {
    return finiteCounter(value.receivedBytes) ?? finiteCounter(value.totalReceivedBytes)
  }
  return finiteCounter(value.sentBytes) ?? finiteCounter(value.totalTransmittedBytes)
}

export function formatNetworkTrafficCounter(
  value: NetworkTrafficCounters | undefined,
  direction: NetworkTrafficDirection,
): string {
  return formatBytes(networkTrafficCounterBytes(value, direction))
}
