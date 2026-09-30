import { readFileSync } from 'node:fs'
import { runInNewContext } from 'node:vm'
import { describe, expect, it, vi } from 'vitest'

const script = readFileSync(new URL('../../public/appearance-init.js', import.meta.url), 'utf8')

function boot(values: Record<string, string> = {}, pathname = '/overview', blocked = false) {
  const properties = new Map<string, string>()
  const classes = new Set<string>()
  const root = { dataset: {} as Record<string, string>, style: { colorScheme: '', setProperty: (key: string, value: string) => properties.set(key, value) }, classList: { add: (name: string) => classes.add(name) } }
  const fetch = vi.fn()
  const Image = vi.fn()
  runInNewContext(script, {
    document: { documentElement: root }, location: { pathname }, matchMedia: () => ({ matches: true }),
    localStorage: { getItem: (key: string) => { if (blocked) throw new Error('blocked'); return values[key] ?? null } },
    sessionStorage: { getItem: (key: string) => values[key] ?? null }, fetch, Image,
  })
  return { root, properties, classes, fetch, Image }
}

describe('appearance before server bootstrap', () => {
  it.each(['light', 'dark'])('restores %s tokens without fetching a cached or default wallpaper', (theme) => {
    const result = boot({ 'kejilion-panel-theme': theme, 'kejilion-panel-desktop-mode': 'desktop', 'kpanel:desktop-wallpaper:v1': 'prism' })
    expect(result.root.dataset.theme).toBe(theme)
    expect(result.root.style.colorScheme).toBe(theme)
    expect(result.classes.has('desktop-boot')).toBe(true)
    expect(result.properties.get('--desktop-wallpaper-image')).toBe('none')
    expect(result.fetch).not.toHaveBeenCalled()
    expect(result.Image).not.toHaveBeenCalled()
  })
  it.each(['/login', '/setup', '/share/token', '/overview'])('does not request stale/private/default images on %s', (path) => {
    const result = boot({ 'kpanel:desktop-wallpaper:v1': 'custom:' + 'a'.repeat(32), 'kpanel:auth-wallpaper:v1': '{"image":"data:image/webp;base64,UklGRg=="}' }, path)
    expect(result.properties.get('--auth-wallpaper-image')).toBe('none')
    expect(result.properties.has('--classic-wallpaper-image')).toBe(false)
    expect(result.fetch).not.toHaveBeenCalled()
    expect(result.Image).not.toHaveBeenCalled()
  })
  it.each(['/login', '/setup', '/share/token'])('does not show the desktop boot layer on %s', (path) => {
    expect(boot({ 'kejilion-panel-desktop-mode': 'desktop' }, path).classes.has('desktop-boot')).toBe(false)
  })
  it('keeps startup available when browser storage is blocked', () => {
    expect(boot({}, '/overview', true).root.dataset.theme).toBe('dark')
  })
  it('accepts matching color tokens but rejects CSS URL injection', () => {
    const cache = JSON.stringify({ theme: 'dark', colors: null, tokens: { '--desktop-wallpaper-veil-dark': 'linear-gradient(145deg, rgb(0 0 0 / 26%), rgb(0 0 0 / 48%))', '--desktop-aurora-one': 'url(https://example.com/x)' } })
    const result = boot({ 'kpanel:desktop-backdrop:v1': cache })
    expect(result.properties.get('--desktop-wallpaper-veil-dark')).toContain('26%')
    expect(result.properties.has('--desktop-aurora-one')).toBe(false)
    expect(boot({ 'kpanel:desktop-backdrop:v1': cache, 'kejilion-panel-colors': 'changed' }).properties.has('--desktop-wallpaper-veil-dark')).toBe(false)
  })
})
