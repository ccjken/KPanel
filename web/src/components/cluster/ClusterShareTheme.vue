<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from '@/i18n'
import { useTheme } from '@/stores/theme'
import { shareThemeModel, shareThemeURL } from '@/lib/shareThemes'
import type { PublicClusterShareSnapshot } from '@/types/api'

const props = defineProps<{ snapshot?: PublicClusterShareSnapshot; errorMessage?: string }>()
const { t, locale } = useI18n()
const { resolved } = useTheme()
const frame = ref<HTMLIFrameElement>()
const ready = ref(false)
const fallback = ref(false)
const failed = ref(false)
const url = computed(() => shareThemeURL(props.snapshot?.theme))
const name = computed(() => props.snapshot?.theme?.name[locale.value] || props.snapshot?.theme?.name['en-US'] || '')
let timer: ReturnType<typeof setTimeout> | undefined
function clearTimer() { if (timer) clearTimeout(timer); timer = undefined }
function stop(error = false) { clearTimer(); fallback.value = true; ready.value = false; failed.value = error }
function send() {
  if (!ready.value || !props.snapshot) return
  frame.value?.contentWindow?.postMessage({ source: 'kpanel-share', type: 'snapshot', schema: 1,
    locale: locale.value, mode: resolved.value, data: shareThemeModel(props.snapshot, locale.value) }, '*')
}
function message(event: MessageEvent) {
  if (!frame.value || event.source !== frame.value.contentWindow || event.origin !== 'null') return
  if (event.data?.source !== 'kpanel-share-theme' || event.data?.type !== 'ready' || ready.value) return
  clearTimer(); ready.value = true; send()
}
watch(url, () => {
  clearTimer(); ready.value = false; fallback.value = false; failed.value = false
  if (url.value) timer = setTimeout(() => stop(true), 12_000)
}, { immediate: true })
watch([() => props.snapshot, locale, resolved], send)
onMounted(() => window.addEventListener('message', message))
onBeforeUnmount(() => { clearTimer(); window.removeEventListener('message', message) })
</script>

<template>
  <p v-if="ready && errorMessage" class="share-theme__notice" role="alert">{{ errorMessage }}</p>
  <section v-if="url && !fallback" class="share-theme" :aria-label="name">
    <div class="share-theme__bar">
      <span>{{ ready ? name : t('cluster.themes.loadingTheme') }}</span>
      <button class="button button--secondary" type="button" @click="stop()">{{ t('cluster.themes.useDefault') }}</button>
    </div>
    <iframe ref="frame" :key="url" :src="url" :title="name" sandbox="allow-scripts" referrerpolicy="no-referrer"
      :class="{ 'is-ready': ready }" :tabindex="ready ? 0 : -1" :aria-hidden="!ready" @error="stop(true)" />
  </section>
  <p v-if="failed" class="share-theme__notice" role="status">{{ t('cluster.themes.fallback') }}</p>
  <slot v-if="!ready" />
</template>

<style scoped>
.share-theme__bar { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 0.75rem; margin-bottom: 1rem; font-size: 0.875rem; color: var(--text-soft); }
iframe { display: none; border: 0; width: 100%; height: max(680px, calc(100dvh - 180px)); background: var(--bg); border-radius: var(--radius-lg); }
iframe.is-ready { display: block; }
.share-theme__notice { font-size: 0.875rem; color: var(--text-soft); }
</style>
