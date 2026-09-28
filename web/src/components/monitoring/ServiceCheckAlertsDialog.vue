<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import LoadingState from '@/components/feedback/LoadingState.vue'
import ErrorState from '@/components/feedback/ErrorState.vue'
import { api, ApiError } from '@/lib/api'
import { phraseCatalogVersion, translatePhrase, usePhraseCatalog } from '@/i18n/phrase'
import type { CheckAlertSnapshot, CheckSubscription } from '@/types/serviceCheckAlerts'

usePhraseCatalog(locale => locale === 'en-US'
  ? import('@/i18n/pages/ServiceCheckAlerts/en-US').then(m => m.default)
  : import('@/i18n/pages/ServiceCheckAlerts/zh-TW').then(m => m.default))
const props = defineProps<{ open: boolean; hostId?: string }>()
const emit = defineEmits<{ close: [] }>()
const snapshot = ref<CheckAlertSnapshot>()
const enabled = ref(false)
const repeat = ref(false)
const subscriptions = ref<CheckSubscription[]>([])
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const saved = ref(false)
let controller: AbortController | undefined
let generation = 0
function phrase(value: string) { phraseCatalogVersion.value; return translatePhrase(value) }
const hosts = computed(() => snapshot.value?.hosts.filter(host => !props.hostId || host.id === props.hostId || props.hostId === 'local' && host.isLocal) || [])
const missing = computed(() => subscriptions.value.filter(sub => (!props.hostId || hosts.value.some(host => host.id === sub.hostId))
  && !snapshot.value?.hosts.find(h => h.id === sub.hostId)?.checks?.items.some(c => c.id === sub.checkId && c.revision === sub.revision)))
function selected(hostId: string, checkId: string, revision: string) {
  return subscriptions.value.some(sub => sub.hostId === hostId && sub.checkId === checkId && sub.revision === revision)
}
function toggle(hostId: string, checkId: string, revision: string, checked: boolean) {
  subscriptions.value = subscriptions.value.filter(sub => sub.hostId !== hostId || sub.checkId !== checkId)
  if (checked) subscriptions.value.push({ hostId, checkId, revision })
  saved.value = false
}
function accept(value: CheckAlertSnapshot) {
  snapshot.value = value; enabled.value = value.enabled; repeat.value = value.repeat
  subscriptions.value = value.subscriptions.map(sub => ({ ...sub }))
}
async function load() {
  controller?.abort(); controller = new AbortController()
  const current = ++generation
  loading.value = true; error.value = ''; saved.value = false
  try {
    const value = await api.cluster.serviceCheckAlerts(controller.signal)
    if (current === generation) accept(value)
  } catch (reason) {
    if (current === generation && !(reason instanceof DOMException && reason.name === 'AbortError')) error.value = phrase('无法读取服务通知设置。')
  } finally { if (current === generation) loading.value = false }
}
async function save() {
  if (!snapshot.value || saving.value || loading.value) return
  saving.value = true; error.value = ''; saved.value = false
  try {
    accept(await api.cluster.updateServiceCheckAlerts({ enabled: enabled.value, repeat: repeat.value,
      subscriptions: subscriptions.value, expectedResourceVersion: snapshot.value.resourceVersion }))
    saved.value = true
  } catch (reason) {
    error.value = reason instanceof ApiError && reason.status === 409
      ? phrase('设置已被修改，请重新读取后再保存。') : phrase('保存失败，当前选择尚未生效。')
  } finally { saving.value = false }
}
watch(() => props.open, open => { if (open) { snapshot.value = undefined; void load() } else { generation++; controller?.abort() } }, { immediate: true })
onBeforeUnmount(() => { generation++; controller?.abort() })
</script>

<template>
  <ModalDialog :open="open" :title="phrase('服务异常通知')" :description="phrase('复用集群通知渠道，按主机选择要关注的检测项。')" size="large" :close-disabled="saving" @close="emit('close')">
    <LoadingState v-if="loading" :message="phrase('正在读取检测项')" />
    <ErrorState v-else-if="error && !snapshot" :title="phrase('读取失败')" :message="error" @retry="load" />
    <div v-else-if="snapshot" class="service-alerts">
      <p class="service-alerts__hint">{{ phrase('连续 3 次失败告警，连续 2 次成功恢复。沿用 5 分钟检测周期，通常需 10–15 分钟确认异常。') }}</p>
      <p v-if="!snapshot.channelReady" class="service-alerts__notice">{{ phrase('请先在集群通知中启用并配置可用渠道。可以先保存订阅。') }}</p>
      <label class="service-alerts__option"><input v-model="enabled" type="checkbox" :disabled="saving" />{{ phrase('启用服务异常通知') }}</label>
      <label class="service-alerts__option"><input v-model="repeat" type="checkbox" :disabled="saving" />{{ phrase('异常持续时每 6 小时提醒一次') }}</label>
      <p class="service-alerts__hint">{{ phrase('主机失联时抑制服务通知；检测数据中断单独提醒。默认不订阅任何检测项。') }}</p>
      <div class="service-alerts__hosts">
        <section v-for="host in hosts" :key="host.id" class="service-alerts__host">
          <h3>{{ host.name || host.id }}</h3>
          <p v-if="!host.checks" class="service-alerts__hint">{{ phrase('暂无检测摘要，请确认节点版本与连接状态。') }}</p>
          <p v-else-if="!host.checks.available" class="service-alerts__notice">{{ phrase('检测暂不可用，已有订阅保留。') }}</p>
          <label v-for="check in host.checks?.items || []" :key="check.id" class="service-alerts__check">
            <input type="checkbox" :checked="selected(host.id, check.id, check.revision)" :disabled="saving" @change="toggle(host.id, check.id, check.revision, ($event.target as HTMLInputElement).checked)" />
            <span><strong>{{ check.name }}</strong><small>{{ check.kind.toUpperCase() }} · {{ check.target }}</small></span>
            <span class="service-alerts__hint">{{ phrase(check.state === 'up' ? '正常' : check.state === 'down' ? '异常' : '等待检测') }}</span>
          </label>
        </section>
        <p v-if="!hosts.length" class="service-alerts__hint">{{ phrase('暂无可订阅主机。') }}</p>
        <div v-for="sub in missing" :key="`${sub.hostId}:${sub.checkId}`" class="service-alerts__check">
          <span>{{ sub.hostId }} / {{ sub.checkId }}<small>{{ phrase('检测已修改、删除或暂不可用，请核对订阅。') }}</small></span>
          <button type="button" class="button button--secondary button--small" :disabled="saving" @click="toggle(sub.hostId, sub.checkId, sub.revision, false)">{{ phrase('移除订阅') }}</button>
        </div>
      </div>
      <p v-if="snapshot.pending" role="status">{{ phrase('待发送通知') }}: {{ snapshot.pending }}</p>
      <p v-if="snapshot.lastError" class="service-alerts__notice" role="alert">{{ phrase('通知发送或存储异常，请检查通知渠道和磁盘状态。') }}</p>
      <p v-if="error" class="service-alerts__notice" role="alert">{{ error }}</p>
      <p v-if="saved" role="status">{{ phrase('服务通知设置已保存。') }}</p>
    </div>
    <template #footer>
      <button class="button button--secondary" type="button" :disabled="saving || loading" @click="load">{{ phrase('重新读取') }}</button>
      <button class="button button--secondary" type="button" :disabled="saving" @click="emit('close')">{{ phrase('关闭') }}</button>
      <button class="button button--primary" type="button" :disabled="saving || loading || !snapshot" @click="save">{{ phrase(saving ? '正在保存…' : '保存设置') }}</button>
    </template>
  </ModalDialog>
</template>

<style scoped>
.service-alerts { display: grid; gap: 14px; color: var(--text); font-size: 14px; }
.service-alerts p { margin: 0; line-height: 1.6; }
.service-alerts__hint, .service-alerts small { color: var(--muted); font-size: 12px; }
.service-alerts__notice { color: var(--danger); }
.service-alerts__option, .service-alerts__check { display: flex; gap: 10px; align-items: center; }
.service-alerts__hosts { display: grid; gap: 14px; max-height: 42vh; overflow-y: auto; }
.service-alerts__host h3 { font-size: 14px; margin: 0 0 8px; }
.service-alerts__check { padding: 10px 0; border-bottom: 1px solid var(--border); }
.service-alerts__check > span:first-of-type { flex: 1; min-width: 0; overflow-wrap: anywhere; }
.service-alerts small { display: block; margin-top: 4px; }
.service-alerts input { flex-shrink: 0; accent-color: var(--accent); }
</style>
