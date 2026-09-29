<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { api } from '@/lib/api'
import { useI18n } from '@/i18n'
import { formatBytes } from '@/lib/format'
import type { ShareThemeList } from '@/lib/shareThemes'
import type { ScenePack } from '@/lib/scenePacks'

const { t, locale } = useI18n()
const catalog = ref<ShareThemeList>()
const busy = ref('')
const loading = ref(true)
const error = ref('')
const notice = ref('')
const controller = new AbortController()
async function load() {
  loading.value = true
  try { catalog.value = await api.cluster.shareThemes(controller.signal); error.value = '' }
  catch { if (!controller.signal.aborted) error.value = t('cluster.themes.loadFailed') }
  finally { loading.value = false }
}
async function act(action: 'install' | 'delete' | 'select', pack?: ScenePack) {
  if (!catalog.value || busy.value) return
  busy.value = `${action}:${pack?.id || 'default'}`; error.value = ''; notice.value = ''
  try {
    if (action === 'install' && pack) await api.cluster.installShareTheme(pack.id, pack.resourceVersion)
    else if (action === 'delete' && pack) await api.cluster.deleteShareTheme(pack.id, pack.resourceVersion)
    else if (action === 'select') await api.cluster.selectShareTheme(pack?.id || '', catalog.value.resourceVersion)
    notice.value = t(action === 'install' ? 'cluster.themes.downloaded' : action === 'delete' ? 'cluster.themes.deleted' : 'cluster.themes.applied')
    await load()
  } catch { error.value = t('cluster.themes.actionFailed'); await loadAfterError() }
  finally { busy.value = '' }
}
async function loadAfterError() {
  try { catalog.value = await api.cluster.shareThemes(controller.signal) } catch { /* Keep actionable retry and last known state. */ }
}
onMounted(load)
onBeforeUnmount(() => controller.abort())
</script>

<template>
  <section class="share-themes" :aria-label="t('cluster.themes.title')" :aria-busy="Boolean(busy) || loading">
    <header><strong>{{ t('cluster.themes.title') }}</strong><a href="https://github.com/kejilion/KPanel/tree/main/share-themes" target="_blank" rel="noopener noreferrer">{{ t('cluster.themes.source') }}</a></header>
    <p class="share-themes__hint">{{ t('cluster.themes.hint') }}</p>
    <div class="share-themes__row">
      <span>{{ t('cluster.themes.default') }} <small>{{ t('cluster.themes.builtin') }}</small></span>
      <span v-if="catalog && !catalog.selected" class="share-themes__current">{{ t('cluster.themes.current') }}</span>
      <button v-else class="button button--secondary" type="button" :disabled="!catalog || Boolean(busy) || loading" @click="act('select')">{{ t('cluster.themes.apply') }}</button>
    </div>
    <div v-for="pack in catalog?.packs || []" :key="pack.id" class="share-themes__row">
      <span class="share-themes__name">{{ pack.name[locale] || pack.name['en-US'] }} <small>{{ formatBytes(pack.sizeBytes) }}</small></span>
      <div class="share-themes__actions">
        <span v-if="catalog?.selected === pack.id" class="share-themes__current">{{ t('cluster.themes.current') }}</span>
        <button v-if="!pack.installed || pack.installedVersion !== pack.version" class="button button--secondary" type="button" :disabled="Boolean(busy) || loading" @click="act('install', pack)">{{ busy === `install:${pack.id}` ? t('cluster.themes.downloading') : t(pack.installed ? 'cluster.themes.update' : 'cluster.themes.download') }}</button>
        <button v-if="pack.installed && catalog?.selected !== pack.id" class="button button--secondary" type="button" :disabled="Boolean(busy) || loading" @click="act('select', pack)">{{ t('cluster.themes.apply') }}</button>
        <button v-if="pack.installed" class="button button--ghost" type="button" :disabled="Boolean(busy) || loading" @click="act('delete', pack)">{{ t('cluster.themes.delete') }}</button>
      </div>
    </div>
    <p v-if="loading" role="status">{{ t('cluster.themes.loading') }}</p>
    <p v-else-if="catalog && !catalog.packs.length">{{ t('cluster.themes.empty') }}</p>
    <p v-if="catalog?.warning" role="status">{{ t('cluster.themes.offline') }}</p>
    <div v-if="error" class="share-themes__error" role="alert">{{ error }} <button class="button button--secondary" type="button" :disabled="Boolean(busy) || loading" @click="load">{{ t('cluster.themes.retry') }}</button></div>
    <p v-else-if="notice" role="status">{{ notice }}</p>
  </section>
</template>

<style scoped>
.share-themes { padding: 1rem; border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface-subtle); font-size: 0.875rem; line-height: 1.5; }
header, .share-themes__row, .share-themes__actions { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 0.5rem 0.75rem; }
header a { font-size: 0.8125rem; color: var(--brand-strong); }
.share-themes__hint, small { color: var(--text-soft); font-size: 0.8125rem; }
.share-themes__hint { margin: 0.375rem 0 0.75rem; }
.share-themes__row { padding: 0.625rem 0; border-top: 1px solid var(--border); }
.share-themes__name { min-width: 0; overflow-wrap: anywhere; }
small { margin-left: 0.5rem; }
.share-themes__actions { justify-content: flex-end; }
.share-themes__current { color: var(--brand-strong); font-size: 0.8125rem; }
.share-themes__error { color: var(--danger); }
.button { font-size: 0.875rem; min-height: 2.25rem; padding: 0.375rem 0.625rem; }
</style>
