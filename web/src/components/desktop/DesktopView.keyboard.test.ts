// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import DesktopView from './DesktopView.vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import { resetDesktopModeForTest, useDesktopMode } from '@/stores/desktopMode'

vi.mock('@/lib/desktopWindowRoute', async (importOriginal) => ({
  ...await importOriginal<typeof import('@/lib/desktopWindowRoute')>(),
  resolveWindowComponent: async () => ({
    template: `<main>
      <input aria-label="First"><input aria-label="Second"><textarea></textarea>
      <select><option>Option</option></select><button>Action</button><a href="#">Link</a>
      <div contenteditable="true"><span>Editable child</span></div>
      <div class="xterm" tabindex="0"></div><div class="cm-editor" tabindex="0"></div>
      <div role="combobox" tabindex="0"></div><iframe title="Embedded app"></iframe>
    </main>`,
  }),
}))

describe('desktop window keyboard selection', () => {
  let wrapper: VueWrapper
  const desktop = useDesktopMode()

  beforeEach(() => {
    window.localStorage.clear()
    resetDesktopModeForTest()
    window.scrollTo = vi.fn()
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1280 })
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 800 })
    desktop.enterDesktop()
  })

  afterEach(() => {
    wrapper?.unmount()
    document.body.innerHTML = ''
    vi.restoreAllMocks()
  })

  async function openWindows(count = 3) {
    const ids = Array.from({ length: count }, () => desktop.openWindow('/settings', 'route.settings', true))
    wrapper = mount(DesktopView, { attachTo: document.body })
    await flushPromises()
    return ids
  }

  function shell(id: number): HTMLElement {
    return wrapper.get(`#desktop-window-${id}`).element as HTMLElement
  }

  async function key(init: KeyboardEventInit = {}) {
    const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true, ...init })
    document.activeElement!.dispatchEvent(event)
    await nextTick()
    return event
  }

  it('cycles three windows in stable order in both directions and moves DOM focus', async () => {
    const ids = await openWindows()
    shell(ids[2]!).focus()
    for (const id of [...ids, ids[0]!]) {
      expect((await key()).defaultPrevented).toBe(true)
      expect(desktop.focusedId.value).toBe(id)
      expect(document.activeElement).toBe(shell(id))
    }
    for (const id of [ids[2]!, ids[1]!, ids[0]!]) {
      await key({ shiftKey: true })
      expect(desktop.focusedId.value).toBe(id)
      expect(document.activeElement).toBe(shell(id))
    }
  })

  it('starts on blank desktop, skips minimized/closing windows and handles close', async () => {
    const ids = await openWindows(4)
    desktop.minimizeWindow(ids[1]!)
    shell(ids[2]!).classList.add('desktop-window--closing')
    ;(wrapper.element as HTMLElement).focus()
    await key()
    expect(desktop.focusedId.value).toBe(ids[0])
    await key()
    expect(desktop.focusedId.value).toBe(ids[3])
    desktop.closeWindow(ids[3]!)
    await nextTick()
    shell(ids[0]!).focus()
    expect((await key()).defaultPrevented).toBe(false)
    expect(desktop.windows.value.find(item => item.id === ids[1])?.minimized).toBe(true)
  })

  it.each([0, 1])('preserves native Tab with %i windows', async (count) => {
    await openWindows(count)
    ;(wrapper.element as HTMLElement).focus()
    expect((await key()).defaultPrevented).toBe(false)
    expect((await key({ shiftKey: true })).defaultPrevented).toBe(false)
  })

  it('preserves Tab and Shift+Tab for all window descendants and desktop buttons', async () => {
    const ids = await openWindows(2)
    const active = ids[1]!
    const controls = shell(active).querySelectorAll<HTMLElement>(
      'input, textarea, select, button, a, [contenteditable], [tabindex], iframe',
    )
    for (const control of [...controls, wrapper.get('.desktop__icon').element as HTMLElement]) {
      control.focus()
      expect(document.activeElement).toBe(control)
      expect((await key()).defaultPrevented).toBe(false)
      expect((await key({ shiftKey: true })).defaultPrevented).toBe(false)
      expect((await key({ key: 'd' })).defaultPrevented).toBe(false)
      expect(desktop.focusedId.value).toBe(active)
      expect(desktop.windows.value.every(item => !item.minimized)).toBe(true)
    }
  })

  it('leaves the window cycle with Enter and resumes after clicking its title', async () => {
    const ids = await openWindows(2)
    shell(ids[1]!).focus()
    await key({ key: 'Enter' })
    expect(document.activeElement).toBe(shell(ids[1]!).querySelector('.desktop-window__action--minimize'))
    expect((await key()).defaultPrevented).toBe(false)
    shell(ids[1]!).querySelector('input')!.focus()
    shell(ids[1]!).querySelector('.desktop-window__titlebar')!.dispatchEvent(new PointerEvent('pointerdown', {
      bubbles: true, button: 0, pointerId: 1, clientX: 200, clientY: 100,
    }))
    expect(document.activeElement).toBe(shell(ids[1]!))
    window.dispatchEvent(new PointerEvent('pointerup', { pointerId: 1 }))
    await key()
    expect(desktop.focusedId.value).toBe(ids[0])
  })

  it('respects modifiers, composition and already handled events', async () => {
    const ids = await openWindows(2)
    shell(ids[1]!).focus()
    for (const init of [{ ctrlKey: true }, { altKey: true }, { metaKey: true }, { isComposing: true }]) {
      expect((await key(init)).defaultPrevented).toBe(false)
      expect((await key({ key: 'd', ...init })).defaultPrevented).toBe(false)
      expect(desktop.focusedId.value).toBe(ids[1])
    }
    const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    event.preventDefault()
    shell(ids[1]!).dispatchEvent(event)
    expect(desktop.focusedId.value).toBe(ids[1])
  })

  it('does not switch behind a modal or an open desktop menu', async () => {
    const ids = await openWindows(2)
    const modal = mount(ModalDialog, { props: { open: true, title: 'Modal' }, attachTo: document.body })
    try {
      await flushPromises()
      await key()
      expect(desktop.focusedId.value).toBe(ids[1])
      expect((await key({ key: 'd' })).defaultPrevented).toBe(false)
      // Also protect the brief interval before a modal takes DOM focus.
      shell(ids[1]!).focus()
      await key()
      expect(desktop.focusedId.value).toBe(ids[1])
    } finally {
      modal.unmount()
    }
    await wrapper.trigger('contextmenu', { button: 2, clientX: 200, clientY: 150 })
    ;(wrapper.element as HTMLElement).focus()
    expect((await key()).defaultPrevented).toBe(false)
    expect((await key({ key: 'd' })).defaultPrevented).toBe(false)
    expect(desktop.focusedId.value).toBe(ids[1])
  })

  it('shows the desktop with D and restores the visible windows in their previous order', async () => {
    const ids = await openWindows(4)
    desktop.minimizeWindow(ids[1]!)
    desktop.toggleMaximize(ids[2]!)
    desktop.focusWindow(ids[0]!)
    await nextTick()
    const before = desktop.windows.value.map(item => ({
      id: item.id, minimized: item.minimized, maximized: item.maximized, geometry: { ...item.geometry },
    }))
    const order = desktop.windows.value.filter(item => !item.minimized).sort((a, b) => a.z - b.z).map(item => item.id)
    shell(ids[0]!).focus()
    expect((await key({ key: 'd' })).defaultPrevented).toBe(true)
    expect(desktop.windows.value.every(item => item.minimized)).toBe(true)
    expect(desktop.focusedId.value).toBe(0)
    expect(document.activeElement).toBe(wrapper.element)
    expect((await key({ key: 'd', repeat: true })).defaultPrevented).toBe(false)
    expect(desktop.focusedId.value).toBe(0)
    // Caps Lock is still a plain D; holding Shift or another modifier is not.
    expect((await key({ key: 'D' })).defaultPrevented).toBe(true)
    expect(desktop.windows.value.map(item => ({
      id: item.id, minimized: item.minimized, maximized: item.maximized, geometry: { ...item.geometry },
    }))).toEqual(before)
    expect(desktop.windows.value.filter(item => !item.minimized).sort((a, b) => a.z - b.z).map(item => item.id)).toEqual(order)
    expect(desktop.focusedId.value).toBe(ids[0])
    expect(document.activeElement).toBe(shell(ids[0]!))
    expect((await key({ key: 'D', shiftKey: true })).defaultPrevented).toBe(false)
    expect(desktop.focusedId.value).toBe(ids[0])
    await key({ key: 'd' })
    await key({ key: 'd' })
    expect(desktop.focusedId.value).toBe(ids[0])
  })

  it('does not reopen windows closed while showing the desktop', async () => {
    const ids = await openWindows(2)
    shell(ids[1]!).focus()
    await key({ key: 'd' })
    desktop.closeWindow(ids[1]!)
    await nextTick()
    await key({ key: 'd' })
    expect(desktop.windows.value).toHaveLength(1)
    expect(desktop.windows.value[0]?.minimized).toBe(false)
    expect(document.activeElement).toBe(shell(ids[0]!))
  })

  it.each(['open', 'restore'])('starts a new cycle after a manual window %s', async (action) => {
    const ids = await openWindows(2)
    shell(ids[1]!).focus()
    await key({ key: 'd' })
    const id = action === 'open' ? desktop.openWindow('/settings', 'route.settings', true) : ids[0]!
    if (action === 'restore') desktop.restoreWindow(id)
    await flushPromises()
    shell(id).focus()
    await key({ key: 'd' })
    expect(desktop.windows.value.every(item => item.minimized)).toBe(true)
    await key({ key: 'd' })
    expect(desktop.windows.value.filter(item => !item.minimized).map(item => item.id)).toEqual([id])
    expect(document.activeElement).toBe(shell(id))
  })

  it.each([0, 1])('leaves D alone with %i already minimized windows and no snapshot', async (count) => {
    const ids = await openWindows(count)
    for (const id of ids) desktop.minimizeWindow(id)
    await nextTick()
    ;(wrapper.element as HTMLElement).focus()
    expect((await key({ key: 'd' })).defaultPrevented).toBe(false)
    expect(desktop.windows.value.every(item => item.minimized)).toBe(true)
  })
})
