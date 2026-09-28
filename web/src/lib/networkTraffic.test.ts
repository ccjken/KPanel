import { describe, expect, it } from 'vitest'
import {
  formatNetworkTrafficCounter,
  networkTrafficCounterBytes,
  clusterTrafficCounters,
  trafficPeriodHint,
} from './networkTraffic'
import { useI18n } from '@/i18n'

describe('network traffic presentation', () => {
  it('selects cycle counters, preserves legacy totals, and never falls back when unavailable', () => {
    const raw = { receivedBytes: 5000, sentBytes: 6000 }
    const period = { receivedBytes: 50, sentBytes: 60, available: true, startedAt: '2026-09-01T00:00:00Z', endsAt: '2026-10-01T00:00:00Z', partial: true, estimated: true }
    expect(clusterTrafficCounters({ network: raw })).toBe(raw)
    expect(clusterTrafficCounters({ lastSnapshot: { telemetry: { network: raw } } })).toBe(raw)
    expect(clusterTrafficCounters({ network: raw, trafficPeriod: period })).toBe(period)
    expect(clusterTrafficCounters({ network: raw, trafficPeriod: { ...period, available: false } })).toBeUndefined()
    const { t } = useI18n()
    expect(trafficPeriodHint(period, t)).toContain('2026-09-01')
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
