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
    expect(wrapper.text()).toBe('价格 $5/月')
    await wrapper.setProps({ details: {} })
    expect(wrapper.find('.host-details').exists()).toBe(false)
  })

  it('uses expiry, price, reset order and renders user text without HTML', () => {
    const wrapper = mount(ClusterHostDetails, { props: { details: {
      expiresOn: '2028-02-29', price: '<b>$5/月</b>', trafficResetDay: 31,
    } } })
    expect(wrapper.findAll('span').map(span => span.text())).toEqual([
      '到期 2028-02-29', '价格 <b>$5/月</b>', '流量每月 31 日重置',
    ])
    expect(wrapper.find('b').exists()).toBe(false)
  })
})
