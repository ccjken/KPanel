<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@/i18n'
import { formatBytes } from '@/lib/format'
import { monthlyTrafficUsage, trafficPeriodHint } from '@/lib/networkTraffic'
import type { ClusterHostDetails, ClusterTrafficPeriod } from '@/types/api'

const props = defineProps<{ period?: ClusterTrafficPeriod; details?: ClusterHostDetails }>()
const { t } = useI18n()
const monthly = computed(() => Boolean(props.period || props.details?.trafficResetDay))
const usage = computed(() => monthlyTrafficUsage(props.period, props.details))
const hint = computed(() => {
  const parts = [trafficPeriodHint(props.period, t)]
  if (monthly.value && !props.period) parts.push(t('cluster.traffic.waiting'))
  if (usage.value) {
    parts.push(t('cluster.traffic.quotaSummary', { used: formatBytes(usage.value.used), quota: formatBytes(usage.value.quotaBytes),
      method: t(`cluster.traffic.calculation.${usage.value.calculation}`) }))
    if (usage.value.percent >= 100) parts.push(t('cluster.traffic.exceeded'))
    else if (usage.value.percent >= 80) parts.push(t('cluster.traffic.nearQuota'))
  }
  return parts.filter(Boolean).join(' · ') || undefined
})
</script>

<template>
  <span class="cluster-traffic-heading" :class="{ 'is-monthly': monthly }" :title="hint">
    {{ t(monthly ? 'cluster.traffic.monthly' : 'cluster.traffic.cumulative') }}<span
      v-if="usage" class="cluster-traffic-heading__usage" :class="`is-${usage.tone}`"
    > · {{ usage.text }}</span>
    <span v-if="hint" class="sr-only"> · {{ hint }}</span>
  </span>
</template>

<style scoped>
.cluster-traffic-heading.is-monthly { color: var(--brand); }
.cluster-traffic-heading .cluster-traffic-heading__usage { font-size: inherit; color: inherit; }
.cluster-traffic-heading .cluster-traffic-heading__usage.is-warning { color: var(--warning); }
.cluster-traffic-heading .cluster-traffic-heading__usage.is-danger { color: var(--danger); }
</style>
