import { describe, expect, it } from 'vitest'
import { sortClusterHostsByDetails, sortClusterHostsTemporarily } from './clusterHostTemporarySort'
import type { ClusterHost } from '@/types/api'

function host(
  id: string,
  cpu: number,
  memory: number,
  disk: number,
  received: number,
  sent: number,
): ClusterHost {
  return {
    id,
    lastSnapshot: {
      telemetry: {
        cpu: { usagePercent: cpu },
        memory: { usagePercent: memory },
        disk: { usagePercent: disk },
        network: { receivedBytes: received, sentBytes: sent },
      },
    },
  } as ClusterHost
}

describe('temporary cluster host sorting', () => {
  const customOrder = [
    host('custom-first', 20, 80, 30, 100, 200),
    host('custom-second', 70, 10, 50, 900, 100),
    host('custom-third', 20, 40, 90, 400, 400),
  ]

  it.each([
    ['cpu', ['custom-second', 'custom-first', 'custom-third']],
    ['memory', ['custom-first', 'custom-third', 'custom-second']],
    ['disk', ['custom-third', 'custom-second', 'custom-first']],
    ['traffic', ['custom-second', 'custom-third', 'custom-first']],
  ] as const)('sorts %s descending and uses custom order for equal values', (key, expected) => {
    expect(sortClusterHostsTemporarily(customOrder, key, 'desc').map((item) => item.id))
      .toEqual(expected)
  })

  it('supports ascending order without mutating the custom order', () => {
    expect(sortClusterHostsTemporarily(customOrder, 'cpu', 'asc').map((item) => item.id))
      .toEqual(['custom-first', 'custom-third', 'custom-second'])
    expect(customOrder.map((item) => item.id))
      .toEqual(['custom-first', 'custom-second', 'custom-third'])
  })

  it('keeps missing and invalid metrics last in either direction', () => {
    const missing = { ...customOrder[0]!, id: 'missing', lastSnapshot: undefined }
    const invalid = host('invalid', Number.NaN, 0, 0, 0, 0)
    const values: ClusterHost[] = [missing, customOrder[1]!, invalid]

    expect(sortClusterHostsTemporarily(values, 'cpu', 'desc').map((item) => item.id))
      .toEqual(['custom-second', 'missing', 'invalid'])
    expect(sortClusterHostsTemporarily(values, 'cpu', 'asc').map((item) => item.id))
      .toEqual(['custom-second', 'missing', 'invalid'])
  })

  it('returns a separate array in the unchanged custom order', () => {
    const result = sortClusterHostsTemporarily(customOrder, 'custom', 'desc')

    expect(result).toEqual(customOrder)
    expect(result).not.toBe(customOrder)
  })

  it('sorts expiry independently of telemetry, leaves missing dates last and preserves equal-date order', () => {
    const items = ['empty', 'late', 'early', 'same', 'invalid'].map(id => ({ id }) as ClusterHost)
    const details = {
      late: { expiresOn: '2028-02-29' }, early: { expiresOn: '2027-01-01' },
      same: { expiresOn: '2027-01-01' }, invalid: { expiresOn: 'invalid' },
    }
    expect(sortClusterHostsTemporarily(items, 'expiresOn', 'asc', details).map(h => h.id))
      .toEqual(['early', 'same', 'late', 'empty', 'invalid'])
    expect(sortClusterHostsTemporarily(items, 'expiresOn', 'desc', details).map(h => h.id))
      .toEqual(['late', 'early', 'same', 'empty', 'invalid'])
  })

  it('compares monthly equivalents within currency and billing groups, including zero', () => {
    const items = [
      { id: 'annual', price: '¥120/年' }, { id: 'monthly', price: 'CNY 12/month' },
      { id: 'dollars', price: '$1/月' }, { id: 'zero', price: '0元/月' },
      { id: 'unknown-cycle', price: '¥9' }, { id: 'missing' }, { id: 'offer', price: '$1–5/月' },
    ]
    expect(sortClusterHostsByDetails(items, 'price', 'asc', h => h).map(h => h.id))
      .toEqual(['zero', 'annual', 'monthly', 'unknown-cycle', 'dollars', 'missing', 'offer'])
    expect(sortClusterHostsByDetails(items, 'price', 'desc', h => h).map(h => h.id))
      .toEqual(['monthly', 'annual', 'zero', 'unknown-cycle', 'dollars', 'missing', 'offer'])
    expect(items[0]!.id).toBe('annual')
  })

  it('recognizes grouped amounts and multi-month cycles while preserving equal monthly prices', () => {
    const items = [
      { price: 'USD 1,200/year' }, { price: '$600/半年' }, { price: '200 USD/2 months' },
      { price: '$299/季' }, { price: '$5/0月' }, { price: '首年$1，续费$10' },
    ]
    expect(sortClusterHostsByDetails(items, 'price', 'asc', h => h).map(h => h.price))
      .toEqual(['$299/季', 'USD 1,200/year', '$600/半年', '200 USD/2 months', '$5/0月', '首年$1，续费$10'])
  })

  it.each(['asc', 'desc'] as const)('preserves equivalent decimal prices across cycles in %s order', direction => {
    for (const prices of [
      ['$0.10/month', '$0.30/3 months', '$0.30/3个月'],
      ['$0.40/month', '$4.80/year'],
      ['$0.0000001/month', '$0.0000012/year'],
    ]) {
      const items = prices.map(price => ({ price }))
      expect(sortClusterHostsByDetails(items, 'price', direction, h => h)).toEqual(items)
    }
  })
})
