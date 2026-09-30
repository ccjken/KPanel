// Run before the application modules, under the existing script-src 'self' policy.
(() => {
  const root = document.documentElement
  const read = key => {
    try { return localStorage.getItem(key) } catch { return null }
  }
  const preference = read('kejilion-panel-theme')
  const systemTheme = matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  const theme = preference === 'light' || preference === 'dark'
    ? preference : systemTheme
  root.dataset.theme = theme
  root.style.colorScheme = theme
  const readSession = key => {
    try { return sessionStorage.getItem(key) } catch { return null }
  }
  // Restore only locally generated gradient/color tokens, never CSS URLs from storage.
  try {
    const raw = readSession('kpanel:desktop-backdrop:v1')
    const cached = raw && raw.length < 12000 ? JSON.parse(raw) : null
    if (cached?.theme === theme && cached.colors === read('kejilion-panel-colors')) {
      for (const token of ['--bg', '--brand-soft', '--desktop-wallpaper-base', '--desktop-wallpaper-veil-light', '--desktop-wallpaper-veil-dark', '--desktop-wallpaper-vignette', '--desktop-aurora-one', '--desktop-aurora-two', '--desktop-aurora-opacity']) {
        const value = cached.tokens?.[token]
        if (typeof value === 'string' && value.length < 1500 && /^[a-zA-Z0-9#.,()% /+-]+$/.test(value) && !/url/i.test(value)) root.style.setProperty(token, value)
      }
    }
  } catch { /* Invalid or unavailable cache leaves the shared theme defaults. */ }
  // Wait for server appearance before requesting a wallpaper. The HTTP cache handles images.
  root.style.setProperty('--desktop-wallpaper-image', 'none')
  root.style.setProperty('--auth-wallpaper-image', 'none')
  const publicPage = /^\/(login|setup|share)(\/|$)/.test(location.pathname)
  if (read('kejilion-panel-desktop-mode') === 'desktop' && !publicPage) {
    root.classList.add('desktop-boot')
  }
})()
