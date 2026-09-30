<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { Image as ImageIcon } from '@lucide/vue'
import { api } from '@/lib/api'
import { acceptAppearanceSnapshot } from '@/lib/appearanceSync'
import { DEFAULT_SITE_ICON } from '@/stores/branding'
import { useI18n } from '@/i18n'
import { prepareSiteIcon } from '@/lib/siteBranding'

const i18n = useI18n()
const form = reactive({ name: '', icon: '' })
let baseline = { name: '', icon: '' }
let resetAll = false
const loading = ref(true)
const loaded = ref(false)
const busy = ref(false)
const error = ref('')
const saved = ref(false)
const controller = new AbortController()
onBeforeUnmount(() => controller.abort())

async function load(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const value = await api.desktop.appearance(controller.signal)
    if (controller.signal.aborted) return
    form.name = value.branding?.name || ''
    form.icon = value.branding?.icon || ''
    baseline = { ...form }
    resetAll = false
    loaded.value = true
  } catch {
    if (!controller.signal.aborted) error.value = i18n.t('branding.loadFailed')
  } finally { loading.value = false }
}

async function chooseIcon(event: Event): Promise<void> {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || busy.value) return
  busy.value = true
  saved.value = false
  error.value = ''
  try {
    const icon = await prepareSiteIcon(file)
    if (!controller.signal.aborted) form.icon = icon
  } catch { error.value = i18n.t('branding.iconInvalid') }
  finally { busy.value = false }
}

function reset(): void {
  resetAll = true
  form.name = ''
  form.icon = ''
  saved.value = false
  error.value = ''
}

async function save(): Promise<void> {
  if (busy.value || !loaded.value) return
  const name = form.name.trim()
  if ([...name].length > 64 || /[\u0000-\u001f\u007f-\u009f]/.test(name)) {
    error.value = i18n.t('branding.nameInvalid')
    return
  }
  busy.value = true
  saved.value = false
  error.value = ''
  try {
    // Merge only explicit edits into current state. Concurrent edits to the
    // same field need a retry; unrelated theme/icon changes stay intact.
    const current = await api.desktop.appearance(controller.signal)
    if (controller.signal.aborted) return
    const latest = { name: current.branding?.name || '', icon: current.branding?.icon || '' }
    const edited = { name, icon: form.icon }
    const changed = { name: resetAll || name !== baseline.name, icon: resetAll || form.icon !== baseline.icon }
    const conflict = (['name', 'icon'] as const).some((key) => changed[key] && latest[key] !== baseline[key] && latest[key] !== edited[key])
    if (conflict) {
      for (const key of ['name', 'icon'] as const) if (!changed[key]) form[key] = latest[key]
      baseline = latest
      throw new Error('Branding changed')
    }
    const value = await api.desktop.updateAppearance({
      theme: current.theme, colors: current.colors, wallpaper: current.wallpaper,
      classicLevel: current.classicLevel, expectedResourceVersion: current.resourceVersion,
      branding: { name: changed.name ? name : latest.name, icon: changed.icon ? form.icon : latest.icon },
    })
    if (controller.signal.aborted) return
    acceptAppearanceSnapshot(value)
    form.name = value.branding?.name || ''
    form.icon = value.branding?.icon || ''
    baseline = { ...form }
    resetAll = false
    saved.value = true
  } catch { if (!controller.signal.aborted) error.value = i18n.t('branding.saveFailed') }
  finally { busy.value = false }
}

onMounted(load)
</script>

<template>
  <section class="settings-section panel-card">
    <header class="settings-section__header">
      <span><ImageIcon :size="19" /></span>
      <div><h2>{{ i18n.t('branding.title') }}</h2><p>{{ i18n.t('branding.description') }}</p></div>
    </header>
    <p v-if="loading" role="status">{{ i18n.t('branding.loading') }}</p>
    <form v-else-if="loaded" class="site-branding-form" @submit.prevent="save">
      <fieldset :disabled="busy">
        <label class="field">
          <span>{{ i18n.t('branding.name') }}</span>
          <input v-model="form.name" name="siteName" placeholder="KPanel" @input="saved = false" />
          <small>{{ i18n.t('branding.nameHint') }}</small>
        </label>
        <div class="site-branding-icon">
          <img :src="form.icon || DEFAULT_SITE_ICON" alt="" width="64" height="64" />
          <label class="field">
            <span>{{ i18n.t('branding.icon') }}</span>
            <input type="file" name="siteIcon" accept="image/png,image/jpeg,image/webp" @change="chooseIcon" />
            <small>{{ i18n.t('branding.iconHint') }}</small>
          </label>
        </div>
        <div class="site-branding-actions">
          <button class="button button--primary" type="submit">{{ i18n.t(busy ? 'branding.saving' : 'branding.save') }}</button>
          <button class="button button--secondary" type="button" @click="reset">{{ i18n.t('branding.reset') }}</button>
        </div>
      </fieldset>
    </form>
    <p v-if="error" class="site-branding-error" role="alert">{{ error }}</p>
    <button v-if="!loading && !loaded" class="button button--secondary" type="button" @click="load">{{ i18n.t('branding.retry') }}</button>
    <p v-if="saved" role="status">{{ i18n.t('branding.saved') }}</p>
  </section>
</template>

<style scoped>
.site-branding-form fieldset { display: grid; gap: 20px; min-width: 0; margin: 0; padding: 0; border: 0; }
.site-branding-form .field { display: grid; gap: 8px; min-width: 0; font-size: 14px; }
.site-branding-form small { color: var(--muted); font-size: 13px; }
.site-branding-icon { display: flex; align-items: center; flex-wrap: wrap; gap: 16px; }
.site-branding-icon img { object-fit: contain; border: 1px solid var(--border); border-radius: var(--radius); }
.site-branding-icon input { max-width: 100%; }
.site-branding-actions { display: flex; flex-wrap: wrap; gap: 12px; }
.site-branding-error { color: var(--danger); }
</style>
