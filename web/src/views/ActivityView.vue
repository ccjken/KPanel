<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AuditView from '@/views/AuditView.vue'
import JobsView from '@/views/JobsView.vue'
import NotificationHistoryView from '@/views/NotificationHistoryView.vue'
import { usePhraseCatalog } from '@/i18n/phrase'

usePhraseCatalog((locale) => locale === 'en-US'
  ? import('@/i18n/pages/ActivityView/en-US').then((module) => module.default)
  : import('@/i18n/pages/ActivityView/zh-TW').then((module) => module.default))

type ActivityTab = 'jobs' | 'audit' | 'notifications'

const route = useRoute()
const router = useRouter()
const activeTab = computed<ActivityTab>(() => (route.query.tab === 'notifications' ? 'notifications' : route.query.tab === 'audit' ? 'audit' : 'jobs'))

function selectTab(tab: ActivityTab): void {
  void router.replace({ path: '/activity', query: { tab } })
}
</script>

<template>
  <div class="page activity-page">
    <div class="activity-page__tabs" role="tablist" aria-label="活动记录类型">
      <button
        type="button"
        role="tab"
        :aria-selected="activeTab === 'jobs'"
        :class="{ 'is-active': activeTab === 'jobs' }"
        @click="selectTab('jobs')"
      >
        变更任务
      </button>
      <button
        type="button"
        role="tab"
        :aria-selected="activeTab === 'audit'"
        :class="{ 'is-active': activeTab === 'audit' }"
        @click="selectTab('audit')"
      >
        安全审计
      </button>
      <button type="button" role="tab" :aria-selected="activeTab === 'notifications'"
        :class="{ 'is-active': activeTab === 'notifications' }" @click="selectTab('notifications')">
        通知记录
      </button>
    </div>
    <component :is="activeTab === 'notifications' ? NotificationHistoryView : activeTab === 'audit' ? AuditView : JobsView"
      :key="activeTab === 'notifications' ? String(route.query.event || '') : activeTab" />
  </div>
</template>

<style scoped>
.activity-page {
  align-content: start;
  gap: 18px;
}

.activity-page > .page {
  row-gap: 12px;
}

.activity-page__tabs {
  display: flex;
  align-self: flex-start;
  flex-wrap: wrap;
  gap: 8px;
}

.activity-page__tabs button {
  min-height: 40px;
  padding: 8px 16px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface-raised);
  color: var(--text-soft);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.5;
  cursor: pointer;
}

.activity-page__tabs button:hover {
  border-color: color-mix(in srgb, var(--brand) 38%, var(--border));
  color: var(--text);
}

.activity-page__tabs button.is-active {
  border-color: color-mix(in srgb, var(--brand) 45%, var(--border));
  background: var(--brand-soft);
  color: var(--brand-strong);
}

.activity-page__tabs button:focus-visible {
  outline: 2px solid var(--brand);
  outline-offset: 2px;
}
</style>
