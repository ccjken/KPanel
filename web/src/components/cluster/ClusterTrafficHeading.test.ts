// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { useI18n } from '@/i18n'
import ClusterTrafficHeading from './ClusterTrafficHeading.vue'

const { t } = useI18n()
const period = { receivedBytes: 29 * 1024 ** 3, sentBytes: 68 * 1024 ** 3, available: true,
  startedAt: '2026-09-15T00:00:00Z', endsAt: '2026-10-15T00:00:00Z', partial: true, estimated: true }

describe('compact traffic heading', () => {
  it('distinguishes unconfigured totals from monthly totals without quota', () => {
    const cumulative = mount(ClusterTrafficHeading)
    expect(cumulative.text()).toBe(t('cluster.traffic.cumulative'))
    expect(cumulative.classes()).not.toContain('is-monthly')
    const monthly = mount(ClusterTrafficHeading, { props: { period } })
    expect(monthly.text()).toContain(t('cluster.traffic.monthly'))
    expect(monthly.classes()).toContain('is-monthly')
    expect(monthly.find('.cluster-traffic-heading__usage').exists()).toBe(false)
  })
  it('shows a single percentage, with quota, mode and incomplete period available in the hint', async () => {
    const wrapper = mount(ClusterTrafficHeading, { props: { period, details: { trafficMonthlyQuotaGiB: 100, trafficCalculation: 'received' } } })
    expect(wrapper.find('.cluster-traffic-heading__usage').text()).toBe('· 29%')
    expect(wrapper.attributes('title')).toContain(t('cluster.traffic.calculation.received'))
    expect(wrapper.attributes('title')).toContain('100.0 GB')
    expect(wrapper.attributes('title')).toContain(t('cluster.traffic.partial'))
    expect(wrapper.attributes('title')).toContain(t('cluster.traffic.estimated'))
    expect(wrapper.html()).not.toContain(period.startedAt)
    expect(wrapper.html()).not.toContain(period.endsAt)
    await wrapper.setProps({ period: { ...period, available: false } })
    expect(wrapper.find('.cluster-traffic-heading__usage').exists()).toBe(false)
    expect(wrapper.attributes('title')).toContain(t('cluster.traffic.waiting'))
  })
  it('marks a newly configured period as waiting without showing a false zero', () => {
    const wrapper = mount(ClusterTrafficHeading, { props: { details: { trafficResetDay: 15, trafficMonthlyQuotaGiB: 100 } } })
    expect(wrapper.classes()).toContain('is-monthly')
    expect(wrapper.find('.cluster-traffic-heading__usage').exists()).toBe(false)
    expect(wrapper.attributes('title')).toContain(t('cluster.traffic.waiting'))
  })
  it.each([[80, 'warning'], [95, 'danger'], [120, 'danger']])('colors %s%% with a readable status hint', (amount, tone) => {
    const wrapper = mount(ClusterTrafficHeading, { props: { period: { ...period, receivedBytes: Number(amount) * 1024 ** 3, sentBytes: 0 }, details: { trafficMonthlyQuotaGiB: 100 } } })
    expect(wrapper.find('.cluster-traffic-heading__usage').classes()).toContain(`is-${tone}`)
    expect(wrapper.attributes('title')).toContain(t(Number(amount) >= 100 ? 'cluster.traffic.exceeded' : 'cluster.traffic.nearQuota'))
  })
})
