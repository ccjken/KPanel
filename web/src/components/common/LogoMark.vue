<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { DEFAULT_SITE_ICON, useSiteBranding } from '@/stores/branding'

const props = withDefaults(
  defineProps<{
    compact?: boolean
    site?: boolean
  }>(),
  {
    compact: false,
    site: false,
  },
)

// A bound public URL avoids test/build tooling treating the root-relative path
// as a local file URL on Windows.
const branding = useSiteBranding()
const failed = ref(false)
const logoPath = computed(() => props.site && !failed.value ? branding.icon.value : DEFAULT_SITE_ICON)
watch(branding.icon, () => { failed.value = false })
</script>

<template>
  <div class="brand" :class="{ 'brand--compact': compact }">
    <span class="brand__mark" aria-hidden="true">
      <img :src="logoPath" alt="" width="38" height="38" @error="failed = true" />
    </span>
    <span v-if="!compact" class="brand__text">
      <strong :title="site ? branding.name.value : undefined">{{ site ? branding.name.value : 'KPanel' }}</strong>
    </span>
  </div>
</template>
