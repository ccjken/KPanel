<script setup lang="ts">
import { useI18n } from '@/i18n'
import type { ClusterHostDetails } from '@/types/api'

defineProps<{ details?: ClusterHostDetails }>()
const { t } = useI18n()
</script>

<template>
  <div v-if="details && (details.expiresOn || details.price || details.trafficResetDay)" class="host-details" :aria-label="t('cluster.details.title')">
    <span v-if="details.expiresOn">{{ t('cluster.details.expirySummary', { date: details.expiresOn }) }}</span>
    <span v-if="details.price">{{ t('cluster.details.priceSummary', { price: details.price }) }}</span>
    <span v-if="details.trafficResetDay">{{ t('cluster.details.resetSummary', { day: details.trafficResetDay }) }}</span>
  </div>
</template>

<style scoped>
.host-details {
  display: flex;
  flex-wrap: wrap;
  gap: 0.2rem 0.65rem;
  margin-top: 0.25rem;
  color: var(--text-soft);
  font-size: 0.8125rem;
  line-height: 1.5;
}
.host-details > span {
  min-width: 0;
  overflow-wrap: anywhere;
}
</style>
