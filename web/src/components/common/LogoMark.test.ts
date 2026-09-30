// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import LogoMark from './LogoMark.vue'
import { applySiteBranding } from '@/stores/branding'

describe('site logo', () => {
  it('updates site identity, preserves product identity and recovers from a broken custom image', async () => {
    applySiteBranding({ name: '<b>My server</b>', icon: 'data:image/png;base64,test' })
    const site = mount(LogoMark, { props: { site: true } })
    const product = mount(LogoMark)
    expect(site.get('strong').text()).toBe('<b>My server</b>')
    expect(site.find('b').exists()).toBe(false)
    expect(product.text()).toBe('KPanel')
    await site.get('img').trigger('error')
    expect(site.get('img').attributes('src')).toBe('/icons/kpanel.svg')
    applySiteBranding({ name: '', icon: 'data:image/png;base64,new' })
    await nextTick()
    expect(site.text()).toBe('KPanel')
    expect(site.get('img').attributes('src')).toContain('base64,new')
    applySiteBranding()
    site.unmount()
    product.unmount()
  })
})
