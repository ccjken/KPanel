<script setup lang="ts">
import { CheckCircle2, CircleAlert, Info, X } from '@lucide/vue'
import { useI18n } from '@/i18n'
import { useToast, type Toast } from '@/stores/toast'

const { items, remove } = useToast()
const i18n = useI18n()
function activate(item: Readonly<Toast>): void {
  remove(item.id)
  item.action?.run()
}
</script>

<template>
  <div class="toast-region" aria-live="polite" aria-atomic="false">
    <TransitionGroup name="toast">
      <article v-for="item in items" :key="item.id" class="toast" :class="[`toast--${item.tone}`, { 'toast--actionable': item.action }]">
        <CheckCircle2 v-if="item.tone === 'success'" :size="19" aria-hidden="true" />
        <CircleAlert v-else-if="item.tone === 'danger'" :size="19" aria-hidden="true" />
        <Info v-else :size="19" aria-hidden="true" />
        <div class="toast__body">
          <strong>{{ item.title }}</strong>
          <p v-if="item.message">{{ item.message }}</p>
          <button v-if="item.action" class="toast__action" type="button" @click="activate(item)">{{ item.action.label }}</button>
        </div>
        <button class="icon-button icon-button--small" type="button" :aria-label="i18n.t('common.closeNotification')" @click="remove(item.id)">
          <X :size="16" />
        </button>
      </article>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-region { max-width: calc(100% - 20px); }
.toast--actionable { max-height: calc(100dvh - 36px); overflow-y: auto; }
.toast--actionable .toast__body { min-width: 0; overflow-wrap: anywhere; }
.toast--actionable .toast__body strong,
.toast--actionable .toast__body p { font-size: 14px; line-height: 1.5; }
.toast--actionable .toast__body p { white-space: pre-line; color: var(--text-soft); }
.toast__action { justify-self: start; min-height: 32px; padding: 4px 0; color: var(--brand); background: none; border: 0; font: inherit; font-size: 14px; cursor: pointer; text-align: start; }
.toast__action:hover { text-decoration: underline; }
.toast__action:focus-visible { outline: 2px solid var(--brand); outline-offset: 3px; border-radius: var(--radius-sm); }
</style>
