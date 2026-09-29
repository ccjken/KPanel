// @vitest-environment jsdom
import { afterEach,describe,expect,it,vi } from 'vitest'
import { flushPromises,mount } from '@vue/test-utils'
import { resetLocaleForTest, setLocale } from '@/i18n'
import AiMarkdown from './AiMarkdown.vue'

afterEach(()=>{vi.unstubAllGlobals();resetLocaleForTest()})

describe('AiMarkdown',()=>{
  it('renders markdown and removes executable HTML',()=>{
    const wrapper=mount(AiMarkdown,{props:{content:'**安全** <img src=x onerror="alert(1)"><script>alert(2)</script>'}})
    expect(wrapper.html()).toContain('<strong>安全</strong>')
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.find('script').exists()).toBe(false)
    expect(wrapper.html()).toContain('&lt;img')
  })

  it('copies fenced code from the top-right action',async()=>{
    const writeText=vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator,'clipboard',{value:{writeText},configurable:true})
    const wrapper=mount(AiMarkdown,{props:{content:'```sh\necho safe\n```'}})
    const button=wrapper.get('button[aria-label="复制代码"]')
    await button.trigger('click')
    expect(writeText).toHaveBeenCalledWith('echo safe\n')
    expect(button.text()).toBe('已复制')
  })

  it.each(['zh-CN','zh-TW','en-US'] as const)('shows a localized copy failure and recovers on retry (%s)',async locale=>{
    await setLocale(locale)
    vi.stubGlobal('navigator',{})
    const fallback=vi.fn().mockReturnValue(false)
    Object.defineProperty(document,'execCommand',{value:fallback,configurable:true})
    const wrapper=mount(AiMarkdown,{props:{content:'```sh\necho safe\n```'}})
    const button=wrapper.get('[data-code-copy]')
    await button.trigger('click');await flushPromises()
    expect(button.text()).toBe('复制')
    expect(wrapper.get('[role="alert"]').text()).toBe({
      'zh-CN':'复制失败，请选中文字后手动复制。',
      'zh-TW':'複製失敗，請選取文字後手動複製。',
      'en-US':'Copy failed. Select the text and copy it manually.',
    }[locale])
    fallback.mockImplementation(()=>{
      expect((document.activeElement as HTMLTextAreaElement).value).toBe('echo safe\n')
      return true
    })
    await button.trigger('click');await flushPromises()
    expect(button.text()).toBe('已复制')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })
})
