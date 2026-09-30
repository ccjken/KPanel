<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { Image as ImageIcon, Upload } from '@lucide/vue'
import { api } from '@/lib/api'
import { saveAppearance } from '@/lib/appearanceSync'
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
const iconInput = ref<HTMLInputElement>()
const iconSelected = ref(false)
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
    iconSelected.value = false
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
    if (!controller.signal.aborted) {
      form.icon = icon
      iconSelected.value = true
    }
  } catch { error.value = i18n.t('branding.iconInvalid') }
  finally { busy.value = false }
}

function reset(): void {
  resetAll = true
  form.name = ''
  form.icon = ''
  iconSelected.value = false
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
    const value = await saveAppearance(async () => {
      const current = await api.desktop.appearance(controller.signal)
      controller.signal.throwIfAborted()
      const latest = { name: current.branding?.name || '', icon: current.branding?.icon || '' }
      const edited = { name, icon: form.icon }
      const changed = { name: resetAll || name !== baseline.name, icon: resetAll || form.icon !== baseline.icon }
      const conflict = (['name', 'icon'] as const).some((key) => changed[key] && latest[key] !== baseline[key] && latest[key] !== edited[key])
      if (conflict) {
        for (const key of ['name', 'icon'] as const) if (!changed[key]) form[key] = latest[key]
        baseline = latest
        throw new Error('Branding changed')
      }
      return api.desktop.updateAppearance({
        theme: current.theme, colors: current.colors, wallpaper: current.wallpaper,
        classicLevel: current.classicLevel, expectedResourceVersion: current.resourceVersion,
        branding: { name: changed.name ? name : latest.name, icon: changed.icon ? form.icon : latest.icon },
      })
    }, controller.signal)
    if (controller.signal.aborted) return
    form.name = value.branding?.name || ''
    form.icon = value.branding?.icon || ''
    iconSelected.value = false
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
    <p v-if="loading" class="site-branding-status" role="status">{{ i18n.t('branding.loading') }}</p>
    <form v-else-if="loaded" class="site-branding-form" @submit.prevent="save">
      <fieldset :disabled="busy">
        <div class="site-branding-fields">
          <label class="field">
            <span>{{ i18n.t('branding.name') }}</span>
            <input v-model="form.name" name="siteName" placeholder="KPanel" @input="saved = false" />
            <small>{{ i18n.t('branding.nameHint') }}</small>
          </label>
          <div class="site-branding-icon-field">
            <span class="site-branding-label">{{ i18n.t('branding.icon') }}</span>
            <div class="site-branding-icon">
              <span class="site-branding-icon__preview">
                <img :src="form.icon || DEFAULT_SITE_ICON" alt="" width="56" height="56" />
              </span>
              <div class="site-branding-icon__controls">
                <button class="button button--secondary" type="button" @click="iconInput?.click()">
                  <Upload :size="16" aria-hidden="true" />
                  {{ i18n.t('branding.changeIcon') }}
                </button>
                <p>{{ i18n.t('branding.iconHint') }}</p>
              </div>
            </div>
            <p class="site-branding-icon__hint" role="status">{{ i18n.t(iconSelected ? 'branding.iconSelected' : 'branding.iconSizeHint') }}</p>
            <input ref="iconInput" class="site-branding-file" type="file" name="siteIcon" accept="image/png,image/jpeg,image/webp" tabindex="-1" aria-hidden="true" @change="chooseIcon" />
          </div>
        </div>
        <div class="site-branding-actions">
          <button class="button button--primary" type="submit">{{ i18n.t(busy ? 'branding.saving' : 'branding.save') }}</button>
          <button class="button button--secondary" type="button" @click="reset">{{ i18n.t('branding.reset') }}</button>
          <span>{{ i18n.t('branding.applyHint') }}</span>
        </div>
      </fieldset>
    </form>
    <p v-if="error" class="site-branding-status site-branding-error" role="alert">{{ error }}</p>
    <button v-if="!loading && !loaded" class="button button--secondary" type="button" @click="load">{{ i18n.t('branding.retry') }}</button>
    <p v-if="saved" class="site-branding-status" role="status">{{ i18n.t('branding.saved') }}</p>
  </section>
</template>

<style scoped>
.site-branding-form { padding: 24px; }
.site-branding-form fieldset { display: grid; gap: 24px; min-width: 0; margin: 0; padding: 0; border: 0; }
.site-branding-fields { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 18rem), 1fr)); align-items: start; gap: 24px 32px; }
.site-branding-form .field { display: grid; gap: 8px; min-width: 0; font-size: 14px; }
.site-branding-form .field > small { color: var(--muted); font-size: 13px; }
.site-branding-form input { font-size: 14px; }
.site-branding-icon-field { display: grid; min-width: 0; gap: 8px; }
.site-branding-label { color: var(--text-soft); font-size: 14px; font-weight: 600; }
.site-branding-icon { display: flex; align-items: center; gap: 16px; }
.site-branding-icon__preview { display: grid; width: 80px; height: 80px; flex: 0 0 auto; place-items: center; background: var(--surface-subtle); border: 1px solid var(--border); border-radius: var(--radius-lg); }
.site-branding-icon__preview img { object-fit: contain; border-radius: var(--radius); }
.site-branding-icon__controls { display: grid; min-width: 0; justify-items: start; gap: 8px; }
.site-branding-icon__controls p, .site-branding-icon__hint { margin: 0; color: var(--muted); font-size: 13px; line-height: 1.5; }
.site-branding-file { display: none; }
.site-branding-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 12px; padding-top: 20px; border-top: 1px solid var(--border); }
.site-branding-actions > span { color: var(--muted); font-size: 13px; line-height: 1.5; }
.site-branding-status { margin: 0; padding: 0 24px 24px; font-size: 14px; line-height: 1.5; }
.site-branding-error { color: var(--danger); }
</style>
