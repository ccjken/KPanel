<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@/i18n'
import { monthlyTrafficUsage, clusterTrafficHint } from '@/lib/networkTraffic'
import type { ClusterHostDetails, ClusterTrafficPeriod } from '@/types/api'

const props = defineProps<{ period?: ClusterTrafficPeriod; details?: ClusterHostDetails }>()
const { t } = useI18n()
const monthly = computed(() => Boolean(props.period || props.details?.trafficResetDay))
const usage = computed(() => monthlyTrafficUsage(props.period, props.details))
const hint = computed(() => clusterTrafficHint(props.period, props.details, t))
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
