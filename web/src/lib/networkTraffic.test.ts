import { describe, expect, it } from 'vitest'
import {
  formatNetworkTrafficCounter,
  networkTrafficCounterBytes,
  clusterTrafficCounters,
  trafficPeriodHint,
  monthlyTrafficUsage,
} from './networkTraffic'
import { useI18n } from '@/i18n'
import type { ClusterTrafficCalculation } from '@/types/api'

const period = { receivedBytes: 29 * 1024 ** 3, sentBytes: 68 * 1024 ** 3, available: true,
  startedAt: '2026-09-15T00:00:00Z', endsAt: '2026-10-15T00:00:00Z', partial: false, estimated: false }

describe('monthly quota usage', () => {
  it.each<[ClusterTrafficCalculation, number]>([['total', 97], ['received', 29], ['sent', 68], ['max', 68]])(
    'uses %s monthly accounting', (trafficCalculation, expected) => {
      expect(monthlyTrafficUsage(period, { trafficMonthlyQuotaGiB: 100, trafficCalculation })?.text).toBe(`${expected}%`)
    },
  )
  it('does not invent a percentage from missing quotas or unavailable counters', () => {
    for (const quota of [undefined, 0, -1, 1.5, 1_048_577, Number.NaN, Infinity]) {
      expect(monthlyTrafficUsage(period, { trafficMonthlyQuotaGiB: quota })).toBeUndefined()
    }
    expect(monthlyTrafficUsage(undefined, { trafficMonthlyQuotaGiB: 100 })).toBeUndefined()
    expect(monthlyTrafficUsage({ ...period, available: false }, { trafficMonthlyQuotaGiB: 100 })).toBeUndefined()
    expect(monthlyTrafficUsage({ ...period, receivedBytes: Number.NaN }, { trafficMonthlyQuotaGiB: 100 })).toBeUndefined()
  })
  it.each([[79.9, '79%', 'normal'], [80, '80%', 'warning'], [94.9, '94%', 'warning'], [95, '95%', 'danger'], [120, '120%', 'danger']])(
    'preserves the %s percent boundary without capping overage', (amount, text, tone) => {
      const usage = monthlyTrafficUsage({ ...period, receivedBytes: Number(amount) * 1024 ** 3, sentBytes: 0 }, { trafficMonthlyQuotaGiB: 100 })
      expect(usage).toMatchObject({ text, tone })
    },
  )
})

describe('network traffic presentation', () => {
  it('selects cycle counters, preserves legacy totals, and never falls back when unavailable', () => {
    const raw = { receivedBytes: 5000, sentBytes: 6000 }
    const period = { receivedBytes: 50, sentBytes: 60, available: true, startedAt: '2026-09-01T00:00:00Z', endsAt: '2026-10-01T00:00:00Z', partial: true, estimated: true }
    expect(clusterTrafficCounters({ network: raw })).toBe(raw)
    expect(clusterTrafficCounters({ lastSnapshot: { telemetry: { network: raw } } })).toBe(raw)
    expect(clusterTrafficCounters({ network: raw, trafficPeriod: period })).toBe(period)
    expect(clusterTrafficCounters({ network: raw, trafficPeriod: { ...period, available: false } })).toBeUndefined()
    const { t } = useI18n()
    expect(trafficPeriodHint(period, t)).not.toContain('2026-09-01')
    expect(trafficPeriodHint(period, t)).not.toContain('2026-10-01')
    expect(trafficPeriodHint(period, t)).toContain(t('cluster.traffic.partial'))
    expect(trafficPeriodHint(period, t)).toContain(t('cluster.traffic.estimated'))
    expect(trafficPeriodHint(undefined, t)).toBeUndefined()
  })
  it('formats host receive/send counters independently', () => {
    const value = { receivedBytes: 17.5 * 1024 ** 3, sentBytes: 3 * 1024 ** 3 }

    expect(formatNetworkTrafficCounter(value, 'received')).toBe('17.5 GB')
    expect(formatNetworkTrafficCounter(value, 'sent')).toBe('3.0 GB')
  })

  it('normalizes the desktop widget field names', () => {
    expect(networkTrafficCounterBytes({ totalReceivedBytes: 1024 }, 'received')).toBe(1024)
    expect(networkTrafficCounterBytes({ totalTransmittedBytes: 2048 }, 'sent')).toBe(2048)
  })

  it('does not turn invalid counters into usage', () => {
    expect(networkTrafficCounterBytes({ receivedBytes: Number.NaN }, 'received')).toBeUndefined()
    expect(networkTrafficCounterBytes({ sentBytes: -1 }, 'sent')).toBeUndefined()
    expect(formatNetworkTrafficCounter({ receivedBytes: Number.NaN }, 'received')).toBe('—')
  })
})
