import { describe, expect, it } from 'vitest'
import { shareThemeModel, shareThemeURL } from './shareThemes'
import type { PublicClusterShareSnapshot } from '@/types/api'
import { useI18n } from '@/i18n'

export function themeSnapshot(): PublicClusterShareSnapshot {
  return { title: 'Public fleet', generatedAt: '2026-01-01T12:00:00Z', total: 1, online: 1, attention: 0,
    theme: { id: 'minimal', name: { 'zh-CN': '简约看板' }, fileBase: `/api/v1/cluster/share-themes/minimal/files/${'b'.repeat(32)}/` },
    items: [{ id: 'public-one', name: '<script>bad</script>', state: 'online', collectedAt: '2026-01-01',
      cpu: { cores: 2, usagePercent: 20 }, memory: { totalBytes: 100, usedBytes: 30, usagePercent: 30 }, disk: { totalBytes: 100, usedBytes: 50, usagePercent: 50 },
      network: { receivedBytes: 1e12, sentBytes: 1e12, receiveBytesPerSecond: 0, transmitBytesPerSecond: 0 },
      location: { country: 'Test' }, load: { one: 0, five: 0, fifteen: 0 },
      trafficPeriod: { available: true, partial: false, estimated: false, receivedBytes: 29 * 1024 ** 3, sentBytes: 68 * 1024 ** 3, startedAt: '2026-01-01', endsAt: '2026-02-01' },
      trafficMonthlyQuotaGiB: 100, trafficCalculation: 'max', price: '$30/月', expiresOn: '2026-01-16' }] }
}
describe('share theme protocol', () => {
  it('accepts only the exact local capability URL', () => {
    const theme = themeSnapshot().theme!
    expect(shareThemeURL(theme)).toBe(`${theme.fileBase}index.html`)
    for (const fileBase of ['https://evil.test/', '//evil.test/', `${theme.fileBase}?token=x`, `${theme.fileBase}../`, theme.fileBase.replace('minimal', 'other')]) {
      expect(shareThemeURL({ ...theme, fileBase })).toBeUndefined()
    }
  })
  it('reuses monthly accounting and value estimates without spreading future API fields', () => {
    const snapshot = themeSnapshot()
    Object.assign(snapshot, { token: 'secret-token', csrf: 'secret-csrf' })
    Object.assign(snapshot.items[0]!, { address: 'secret-address', reminderEnabled: true })
    const model = shareThemeModel(snapshot, 'en-US', new Date(2026, 0, 1, 12))
    expect(model.hosts[0]?.traffic).toMatchObject({ monthly: true, percent: '68%', tone: 'normal' })
    expect(model.value.groups).toEqual([{ currency: 'USD', text: '$15.00' }])
    expect(JSON.stringify(model)).not.toContain('secret-')
    expect(JSON.stringify(model)).not.toContain('reminderEnabled')
    snapshot.items[0]!.trafficPeriod!.available = false
    const waiting = shareThemeModel(snapshot, 'en-US')
    expect(waiting.hosts[0]?.traffic.percent).toBe('')
    expect(waiting.hosts[0]?.traffic.received).toBe('—')
  })
  it('preserves core incomplete/estimated/waiting and accounting explanations', () => {
    const snapshot = themeSnapshot(), { t } = useI18n()
    snapshot.items[0]!.trafficPeriod!.partial = true
    snapshot.items[0]!.trafficPeriod!.estimated = true
    const traffic = shareThemeModel(snapshot, 'zh-CN', new Date(), t).hosts[0]!.traffic
    expect(traffic).toMatchObject({ partial: true, estimated: true })
    expect(traffic.hint).toContain('统计不完整')
    expect(traffic.hint).toContain('按时间比例估算')
    expect(traffic.hint).toContain('收发取较大值')
    snapshot.items[0]!.trafficPeriod!.available = false
    expect(shareThemeModel(snapshot, 'zh-CN', new Date(), t).hosts[0]!.traffic.hint).toContain('等待有效采样')
    delete snapshot.items[0]!.trafficPeriod
    snapshot.items[0]!.trafficResetDay = 1
    expect(shareThemeModel(snapshot, 'zh-CN', new Date(), t).hosts[0]!.traffic.monthly).toBe(true)
    expect(shareThemeModel(snapshot, 'zh-CN', new Date(), t).hosts[0]!.traffic.hint).toContain('等待有效采样')
  })
})
