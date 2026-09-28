<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Bell, RefreshCw, Search } from '@lucide/vue'
import EmptyState from '@/components/feedback/EmptyState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import { ApiError, api } from '@/lib/api'
import { formatDateTime } from '@/lib/format'
import type { NotificationEvent, NotificationHistoryPage } from '@/types/api'
import { phraseCatalogVersion, translatePhrase, usePhraseCatalog } from '@/i18n/phrase'

usePhraseCatalog((locale) => locale === 'en-US'
  ? import('@/i18n/pages/NotificationHistoryView/en-US').then((module) => module.default)
  : import('@/i18n/pages/NotificationHistoryView/zh-TW').then((module) => module.default))
function phrase(value: string): string { phraseCatalogVersion.value; return translatePhrase(value) }

const route = useRoute()
const filters = reactive({ days: '7', host: typeof route.query.host === 'string' ? route.query.host : '', rule: '', kind: '', delivery: '', search: '' })
const items = ref<NotificationEvent[]>([])
const hosts = ref<NotificationHistoryPage['hosts']>([])
const nextCursor = ref('')
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')
const retention = ref({ days: 30, events: 2000 })
let controller: AbortController | undefined
let requestID = 0
let appliedFilters: Record<string, string> = {}

const rules: Record<string, string> = {
  cpu: 'CPU 使用率', memory: '内存使用率', disk: '磁盘使用率', traffic: '网络吞吐',
  'traffic-total-received': '累计接收', 'traffic-total-sent': '累计传送', availability: '主机连接', ssh: 'SSH 登录',
}
const kinds: Record<NotificationEvent['kind'], string> = { alert: '告警', recovery: '恢复', info: '信息' }
const deliveries: Record<NotificationEvent['delivery'], string> = { local_only: '仅本地', pending: '待发送', sent: '已发送', failed: '发送失败', cancelled: '已停止发送' }

function compactMessage(message: string): string {
  return message.replace(/\r?\n(?:[\t ]*\r?\n)+/g, '\n').trim()
}

async function load(append = false): Promise<void> {
  controller?.abort()
  controller = new AbortController()
  const id = ++requestID
  loading.value = !append
  loadingMore.value = append
  error.value = ''
  if (!append) {
    items.value = []
    nextCursor.value = ''
    appliedFilters = { since: new Date(Date.now() - Number(filters.days) * 86400000).toISOString() }
    for (const key of ['host', 'rule', 'kind', 'delivery', 'search'] as const) {
      const value = filters[key].trim()
      if (value) appliedFilters[key] = value
    }
  }
  try {
    const result = await api.cluster.notificationHistory({ ...appliedFilters, ...(append ? { cursor: nextCursor.value } : {}) }, controller.signal)
    if (id !== requestID) return
    items.value = append ? [...items.value, ...result.items] : result.items
    hosts.value = result.hosts
    nextCursor.value = result.nextCursor || ''
    retention.value = { days: result.retentionDays, events: result.maxEvents }
  } catch (reason) {
    if (id !== requestID || (reason instanceof DOMException && reason.name === 'AbortError')) return
    error.value = reason instanceof ApiError && reason.status === 401
      ? '登录已过期，请重新登录。'
      : reason instanceof ApiError && reason.status === 400
        ? '筛选条件无效，请缩短搜索内容或调整筛选后重试。'
        : '通知记录暂时不可用，请重试或检查 KPanel 数据目录。'
  } finally {
    if (id === requestID) { loading.value = false; loadingMore.value = false }
  }
}

watch(() => [filters.days, filters.host, filters.rule, filters.kind, filters.delivery], () => void load())
watch(() => route.query.host, (host) => { filters.host = typeof host === 'string' ? host : '' })
onMounted(() => void load())
onBeforeUnmount(() => { requestID++; controller?.abort() })
</script>

<template>
  <div class="page notification-history">
    <p class="notification-history__intro">{{ phrase('本机与集群事件默认保存在当前 KPanel，外部推送可选。') }}</p>
    <form class="notification-history__filters toolbar-card" @submit.prevent="load()">
      <label class="field notification-history__search">
        <span>{{ phrase('搜索记录') }}</span>
        <div><Search :size="17" aria-hidden="true" /><input v-model="filters.search" type="search" maxlength="200" :placeholder="phrase('主机名称或通知内容')" /></div>
      </label>
      <label class="field"><span>{{ phrase('时间范围') }}</span><select v-model="filters.days">
        <option value="1">{{ phrase('最近 24 小时') }}</option><option value="7">{{ phrase('最近 7 天') }}</option><option value="30">{{ phrase('最近 30 天') }}</option>
      </select></label>
      <label class="field"><span>{{ phrase('主机') }}</span><select v-model="filters.host">
        <option value="">{{ phrase('全部主机') }}</option><option value="local">{{ phrase('仅本机') }}</option>
        <option v-if="filters.host && filters.host !== 'local' && !hosts.some(host => host.id === filters.host)" :value="filters.host">{{ filters.host }}</option>
        <option v-for="host in hosts.filter(host => !host.isLocal)" :key="host.id" :value="host.id">{{ host.name }}</option>
      </select></label>
      <label class="field"><span>{{ phrase('事件类型') }}</span><select v-model="filters.rule">
        <option value="">{{ phrase('全部类型') }}</option><option v-for="(label, key) in rules" :key="key" :value="key">{{ phrase(label) }}</option>
      </select></label>
      <button type="submit" class="button button--secondary" :disabled="loading"><RefreshCw :size="16" />{{ phrase('查询') }}</button>
      <details class="notification-history__more">
        <summary>{{ phrase('更多筛选') }}</summary>
        <div>
          <label class="field"><span>{{ phrase('事件性质') }}</span><select v-model="filters.kind"><option value="">{{ phrase('全部') }}</option><option v-for="(label, key) in kinds" :key="key" :value="key">{{ phrase(label) }}</option></select></label>
          <label class="field"><span>{{ phrase('外部投递') }}</span><select v-model="filters.delivery"><option value="">{{ phrase('全部') }}</option><option v-for="(label, key) in deliveries" :key="key" :value="key">{{ phrase(label) }}</option></select></label>
        </div>
      </details>
    </form>
    <p class="notification-history__retention">{{ phrase('保留最近') }} {{ retention.days }} {{ phrase('天，最多') }} {{ retention.events }} {{ phrase('条；达到容量上限时清理最早记录。') }}</p>
    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" :message="phrase(error)" @retry="load(items.length > 0)" />
    <EmptyState v-else-if="!items.length" :title="phrase('暂无符合条件的通知')" :description="phrase('可以调整筛选条件；首次启用后只记录新发生的事件。')" />
    <div v-if="!loading && items.length" class="notification-history__list">
      <article v-for="event in items" :key="event.id" class="notification-history__event" :data-kind="event.kind" :aria-label="[event.hostName, phrase(rules[event.rule] || event.rule), phrase(kinds[event.kind]), formatDateTime(event.createdAt)].join(' · ')">
        <header class="notification-history__heading">
          <Bell :size="18" aria-hidden="true" />
          <div class="notification-history__subject"><strong>{{ event.hostName }}</strong><span>{{ phrase(rules[event.rule] || event.rule) }}</span></div>
          <span class="notification-history__kind">{{ phrase(kinds[event.kind]) }}</span>
        </header>
        <div class="notification-history__detail">
          <p>{{ compactMessage(event.message) }}</p>
        </div>
        <footer class="notification-history__meta">
          <span :class="{ 'notification-history__failure': event.delivery === 'failed' }">{{ phrase(deliveries[event.delivery]) }}</span>
          <time :datetime="event.createdAt">{{ formatDateTime(event.createdAt) }}</time>
          <span v-if="event.relatedEventId">{{ phrase('关联告警编号') }}: {{ event.relatedEventId }}</span>
          <span v-if="event.provider">{{ phrase('渠道') }}: {{ event.provider }} · {{ phrase('发送次数') }}: {{ event.attempts }}</span>
          <span v-if="event.lastAttemptAt">{{ phrase('最近尝试') }}: {{ formatDateTime(event.lastAttemptAt) }}</span>
          <p v-if="event.delivery === 'failed'" class="notification-history__notice notification-history__failure">{{ phrase('发送失败会自动重试，最长 24 小时；本地记录已保存。') }}</p>
          <p v-if="event.delivery === 'cancelled'" class="notification-history__notice">{{ phrase('外部推送已关闭、配置已变化或重试已到期，本地记录仍保留。') }}</p>
        </footer>
      </article>
      <button v-if="nextCursor" class="button button--secondary" type="button" :disabled="loadingMore" @click="load(true)">{{ phrase(loadingMore ? '正在加载…' : '加载更多') }}</button>
    </div>
  </div>
</template>

<style scoped>
.notification-history { min-width: 0; container: notification-history / inline-size; }
.notification-history__intro { margin: 0; color: var(--text-soft); font-size: 14px; line-height: 1.6; }
.notification-history__filters { display: flex; flex-wrap: wrap; gap: 16px; align-items: end; }
.notification-history__filters label { display: grid; gap: 6px; flex: 1 1 145px; min-width: 0; font-size: 14px; }
.notification-history__filters input, .notification-history__filters select { width: 100%; min-width: 0; min-height: 40px; font-size: 14px; }
.notification-history__filters .notification-history__search { flex-basis: 240px; }
.notification-history__search > div { display: flex; align-items: center; gap: 8px; }
.notification-history__more { flex: 1 0 100%; font-size: 14px; }
.notification-history__more > div { display: flex; flex-wrap: wrap; gap: 16px; margin-top: 12px; }
.notification-history__more summary { cursor: pointer; width: fit-content; padding: 4px 0; }
.notification-history__retention { color: var(--muted); font-size: 13px; line-height: 1.5; margin: 0; }
.notification-history__list { display: grid; grid-template-columns: minmax(0, 1fr); gap: 12px; min-width: 0; }
.notification-history__list > button { grid-column: 1 / -1; justify-self: center; }
.notification-history__event { --event-tone: var(--brand); display: flex; flex-direction: column; background: var(--surface); border: 1px solid var(--border); border-inline-start: 3px solid var(--event-tone); border-radius: var(--radius); min-width: 0; }
.notification-history__event[data-kind="alert"] { --event-tone: var(--danger); }
.notification-history__event[data-kind="recovery"] { --event-tone: var(--success); }
.notification-history__heading { display: flex; gap: 8px; align-items: center; padding: 12px 16px; font-size: 14px; line-height: 1.5; }
.notification-history__heading > svg { flex: none; color: var(--event-tone); }
.notification-history__subject { display: grid; gap: 2px; flex: 1; min-width: 0; overflow-wrap: anywhere; }
.notification-history__subject > span { color: var(--text-soft); font-size: 13px; }
.notification-history__kind { flex: none; color: var(--event-tone); font-weight: 600; }
.notification-history__failure { color: var(--danger); }
.notification-history__detail { padding: 0 16px 12px; font-size: 14px; line-height: 1.6; overflow-wrap: anywhere; }
.notification-history__detail p { white-space: pre-wrap; margin: 0; }
.notification-history__meta { display: flex; flex-wrap: wrap; gap: 4px 12px; margin-top: auto; padding: 10px 16px; border-top: 1px solid var(--border); color: var(--text-soft); font-size: 13px; line-height: 1.5; overflow-wrap: anywhere; }
.notification-history__notice { flex-basis: 100%; margin: 4px 0 0; font-size: 14px; }
.notification-history summary:focus-visible { outline: 2px solid var(--brand); outline-offset: 3px; border-radius: var(--radius-sm); }
@container notification-history (min-width: 740px) { .notification-history__list { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
@container notification-history (min-width: 1112px) { .notification-history__list { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
@container notification-history (min-width: 1484px) { .notification-history__list { grid-template-columns: repeat(4, minmax(0, 1fr)); } }
</style>
