// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ClusterHostDetails from './ClusterHostDetails.vue'

describe('ClusterHostDetails', () => {
  it('omits unset fields and removes the entire line when empty', async () => {
    const wrapper = mount(ClusterHostDetails)
    expect(wrapper.find('.host-details').exists()).toBe(false)
    await wrapper.setProps({ details: { price: '$5/月' } })
    expect(wrapper.findAll('span')).toHaveLength(1)
    expect(wrapper.text()).toBe('$5/月')
    expect(wrapper.get('[role="img"]').attributes('title')).toBe('价格 $5/月')
    await wrapper.setProps({ details: {} })
    expect(wrapper.find('.host-details').exists()).toBe(false)
    await wrapper.setProps({ details: { trafficResetDay: 31 } })
    expect(wrapper.find('.host-details').exists()).toBe(false)
  })

  it('shows expiry and price without the reset date and renders user text without HTML', () => {
    const wrapper = mount(ClusterHostDetails, { props: { details: {
      expiresOn: '2028-02-29', price: '<b>$5/月</b>', trafficResetDay: 31,
    } } })
    expect(wrapper.findAll('span').map(span => span.text())).toEqual([
      '2028-02-29', '<b>$5/月</b>',
    ])
    expect(wrapper.findAll('[role="img"]').map(pill => pill.attributes('aria-label'))).toEqual([
      '到期 2028-02-29', '价格 <b>$5/月</b>',
    ])
    expect(wrapper.find('b').exists()).toBe(false)
  })
})
