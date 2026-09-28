import { clusterTrafficCounters, networkTrafficCounterBytes, type NetworkTrafficCounters } from '@/lib/networkTraffic'
import type { ClusterHost, ClusterHostDetails, PublicClusterShareHost } from '@/types/api'

export type ClusterHostDetailsSortKey = 'custom' | 'expiresOn' | 'price'
export type ClusterHostTemporarySortKey = ClusterHostDetailsSortKey | 'cpu' | 'memory' | 'disk' | 'traffic'
export type ClusterHostTemporarySortDirection = 'asc' | 'desc'

interface SortMetric {
  value: number
  group?: string
  fraction?: { numerator: bigint; denominator: bigint }
}

function priceMetric(price: string | undefined): SortMetric | undefined {
  // Parse a single amount only; offers, ranges and unknown cycles stay unsorted.
  const match = price?.normalize('NFKC').trim().match(/^(?:([a-z]{3}|US\$|HK\$|[¥$€£])\s*)?((?:\d{1,3}(?:,\d{3})+|\d+)(?:\.\d+)?)\s*([a-z]{3}|元|美元|欧元|歐元|英镑|英鎊)?(?:\s*(?:\/|每|per\s+)\s*(\d+)?\s*(月|个月|個月|季|季度|半年|年|mo|month|months|quarter|quarters|yr|year|years))?$/i)
  if (!match || (match[1] && match[3])) return undefined
  const currencies: Record<string, string> = {
    '¥': 'CNY', RMB: 'CNY', 元: 'CNY', '$': 'USD', 'US$': 'USD', 美元: 'USD',
    'HK$': 'HKD', '€': 'EUR', 欧元: 'EUR', 歐元: 'EUR', '£': 'GBP', 英镑: 'GBP', 英鎊: 'GBP',
  }
  const unit = (match[1] || match[3] || '').toUpperCase()
  const currency = currencies[unit] || unit
  const months: Record<string, number> = {
    月: 1, 个月: 1, 個月: 1, mo: 1, month: 1, months: 1, 季: 3, 季度: 3, quarter: 3, quarters: 3,
    半年: 6, 年: 12, yr: 12, year: 12, years: 12,
  }
  const period = match[5]?.toLowerCase()
  const count = Number(match[4] || 1)
  if (!Number.isFinite(count) || count <= 0) return undefined
  const decimal = match[2]!.replaceAll(',', '')
  const amount = Number(decimal)
  const value = period ? amount / (months[period]! * count) : amount
  if (!Number.isFinite(value)) return undefined
  // No exchange rates are assumed; amounts without a cycle form a separate group.
  return {
    value, group: `${currency}:${period ? 'monthly' : 'unspecified'}`,
    fraction: {
      numerator: BigInt(decimal.replace('.', '')),
      denominator: 10n ** BigInt(decimal.split('.')[1]?.length || 0)
        * (period ? BigInt(months[period]!) * BigInt(match[4] || 1) : 1n),
    },
  }
}

function sortByMetric<T>(items: readonly T[], direction: ClusterHostTemporarySortDirection, metric: (host: T) => SortMetric | undefined): T[] {
  return items.map((host, customIndex) => ({ host, customIndex, metric: metric(host) }))
    .sort((left, right) => {
      if (!left.metric && !right.metric) return left.customIndex - right.customIndex
      if (!left.metric) return 1
      if (!right.metric) return -1
      const leftGroup = left.metric.group || ''
      const rightGroup = right.metric.group || ''
      if (leftGroup !== rightGroup) return leftGroup < rightGroup ? -1 : 1
      let order = left.metric.value - right.metric.value
      if (left.metric.fraction && right.metric.fraction) {
        // Preserve equal decimal prices across billing cycles without floating-point drift.
        const difference = left.metric.fraction.numerator * right.metric.fraction.denominator
          - right.metric.fraction.numerator * left.metric.fraction.denominator
        order = difference < 0n ? -1 : Number(difference > 0n)
      }
      return (direction === 'desc' ? -order : order) || left.customIndex - right.customIndex
    }).map(({ host }) => host)
}

export function sortClusterHostsByDetails<T>(
  items: readonly T[], key: ClusterHostDetailsSortKey, direction: ClusterHostTemporarySortDirection,
  details: (host: T) => ClusterHostDetails | undefined,
): T[] {
  if (key === 'custom') return [...items]
  return sortByMetric(items, direction, host => {
    const value = details(host)
    if (key === 'price') return priceMetric(value?.price)
    const date = value?.expiresOn
    if (!date || !/^\d{4}-\d{2}-\d{2}$/.test(date)) return undefined
    const timestamp = Date.parse(`${date}T00:00:00Z`)
    return Number.isFinite(timestamp) ? { value: timestamp } : undefined
  })
}

function normalizedMetric(value: number | undefined): number | undefined {
  if (value === undefined || !Number.isFinite(value)) return undefined
  return Math.max(0, value)
}

function telemetryMetric(
  telemetry: {
    cpu: { usagePercent: number }
    memory: { usagePercent: number }
    disk: { usagePercent: number }
    network: NetworkTrafficCounters
  } | undefined,
  key: ClusterHostTemporarySortKey,
  traffic?: NetworkTrafficCounters,
): number | undefined {
  if (!telemetry || key === 'custom') return undefined
  if (key === 'cpu') return normalizedMetric(telemetry.cpu.usagePercent)
  if (key === 'memory') return normalizedMetric(telemetry.memory.usagePercent)
  if (key === 'disk') return normalizedMetric(telemetry.disk.usagePercent)

  const received = networkTrafficCounterBytes(traffic, 'received')
  const sent = networkTrafficCounterBytes(traffic, 'sent')
  if (received === undefined || sent === undefined) return undefined
  return normalizedMetric(received + sent)
}

export function sortClusterHostsTemporarily(
  items: readonly ClusterHost[],
  key: ClusterHostTemporarySortKey,
  direction: ClusterHostTemporarySortDirection,
  details: Readonly<Record<string, ClusterHostDetails>> = {},
): ClusterHost[] {
  if (key === 'custom') return [...items]
  if (key === 'expiresOn' || key === 'price') return sortClusterHostsByDetails(items, key, direction, host => details[host.id])
  return sortByMetric(items, direction, host => {
    const value = telemetryMetric(host.lastSnapshot?.telemetry, key, clusterTrafficCounters(host))
    return value === undefined ? undefined : { value }
  })
}

export function sortPublicClusterHostsTemporarily(
  items: readonly PublicClusterShareHost[],
  key: ClusterHostTemporarySortKey,
  direction: ClusterHostTemporarySortDirection,
): PublicClusterShareHost[] {
  if (key === 'custom' || key === 'expiresOn' || key === 'price') {
    return sortClusterHostsByDetails(items, key, direction, host => host)
  }
  return sortByMetric(items, direction, host => {
    const value = telemetryMetric(host.collectedAt ? host : undefined, key, clusterTrafficCounters(host))
    return value === undefined ? undefined : { value }
  })
}
