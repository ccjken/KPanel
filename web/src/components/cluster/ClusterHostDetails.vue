<script setup lang="ts">
import { CalendarClock, Coins } from '@lucide/vue'
import { useI18n } from '@/i18n'
import type { ClusterHostDetails } from '@/types/api'

defineProps<{ details?: ClusterHostDetails }>()
const { t } = useI18n()
</script>

<template>
  <div v-if="details && (details.expiresOn || details.price)" class="host-details" :aria-label="t('cluster.details.title')">
    <span v-if="details.expiresOn" class="host-details__pill host-details__pill--expiry" role="img" :title="t('cluster.details.expirySummary', { date: details.expiresOn })" :aria-label="t('cluster.details.expirySummary', { date: details.expiresOn })">
      <CalendarClock :size="13" aria-hidden="true" />{{ details.expiresOn }}
    </span>
    <span v-if="details.price" class="host-details__pill host-details__pill--price" role="img" :title="t('cluster.details.priceSummary', { price: details.price })" :aria-label="t('cluster.details.priceSummary', { price: details.price })">
      <Coins :size="13" aria-hidden="true" />{{ details.price }}
    </span>
  </div>
</template>

<style scoped>
.host-details {
  display: flex;
  flex-wrap: wrap;
  gap: 0.3rem 0.4rem;
  margin-top: 0.4rem;
  color: var(--text-soft);
  font-size: 0.8125rem;
  line-height: 1.5;
}
.host-details__pill {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  min-width: 0;
  padding: 0.125rem 0.5rem;
  color: var(--detail-color);
  border: 1px solid color-mix(in srgb, var(--detail-color) 24%, transparent);
  border-radius: 999px;
  background: color-mix(in srgb, var(--detail-color) 8%, var(--surface));
  overflow-wrap: anywhere;
}
.host-details__pill > svg { flex: none; }
.host-details__pill--expiry { --detail-color: var(--violet); }
.host-details__pill--price { --detail-color: var(--success); }
</style>
