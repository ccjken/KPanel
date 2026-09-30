import { computed, readonly, ref } from 'vue'
import type { SiteBranding } from '@/types/api'

export const DEFAULT_SITE_ICON = '/icons/kpanel.svg'
export const PROJECT_SOURCE_URL = 'https://github.com/kejilion/KPanel'
const branding = ref<SiteBranding>({ name: '', icon: '' })

export function applySiteBranding(value?: SiteBranding): void {
  branding.value = { name: value?.name || '', icon: value?.icon || '' }
}

export function useSiteBranding() {
  return {
    value: readonly(branding),
    name: computed(() => branding.value.name || 'KPanel'),
    icon: computed(() => branding.value.icon || DEFAULT_SITE_ICON),
  }
}
