<script setup lang="ts">
import { computed, ref } from 'vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import { backups, backupMessage, type BackupSettings, type BackupStorage } from '@/lib/backup'
import { phraseCatalogVersion, translatePhrase } from '@/i18n/phrase'
const props = defineProps<{ settings: BackupSettings }>()
const emit = defineEmits<{ close: []; saved: [settings: BackupSettings] }>()
function phrase(value: string) { phraseCatalogVersion.value; return translatePhrase(value) }
const editing = ref<BackupStorage>()
const removing = ref<BackupStorage>()
const busy = ref(false)
const error = ref('')
const notice = ref('')
const valid = computed(() => !!editing.value?.name.trim() && !!editing.value?.endpoint.trim() && (editing.value?.hasSecret || !!editing.value?.secret) && (editing.value.kind === 's3' ? !!editing.value.bucket && !!editing.value.accessKey : !!editing.value.username))
function edit(storage?: BackupStorage) {
  editing.value = storage ? { ...storage, secret: '' } : { id: '', name: '', kind: 's3', endpoint: '', prefix: 'kpanel', region: 'us-east-1', pathStyle: false, secret: '' }
  error.value = ''; notice.value = ''; removing.value = undefined
}
async function perform(task: () => Promise<void>) {
  if (busy.value) return
  busy.value = true; error.value = ''; notice.value = ''
  try { await task() } catch (reason) { error.value = backupMessage(reason) }
  finally { busy.value = false }
}
function save() {
  if (!valid.value || !editing.value) return
  return perform(async () => { emit('saved', await backups.saveStorage(props.settings.revision, editing.value!)); editing.value = undefined; notice.value = '远程存储已保存。' })
}
function test(id: string) { return perform(async () => { await backups.testStorage(id); notice.value = '连接正常，已验证上传、读取、列出和删除权限。' }) }
function remove() { if (removing.value) return perform(async () => { emit('saved', await backups.deleteStorage(props.settings.revision, removing.value!.id)); removing.value = undefined }) }
</script>

<template>
  <ModalDialog :open="true" :title="phrase('远程存储')" size="medium" :close-disabled="busy" @close="emit('close')">
    <div class="backup-options">
      <template v-if="!editing">
        <p class="backup-hint">{{ phrase('连接 S3 兼容存储或 WebDAV，备份文件由面板直接传输。') }}</p>
        <p v-if="!settings.storages.length" class="backup-hint">{{ phrase('还没有远程存储') }}</p>
        <ul class="backup-storage-list">
          <li v-for="storage in settings.storages" :key="storage.id">
            <div><strong>{{ storage.name }}</strong><small>{{ storage.kind === 's3' ? 'S3' : 'WebDAV' }} · {{ storage.prefix || '/' }}</small></div>
            <div class="backup-option-actions">
              <button type="button" class="button button--secondary" :disabled="busy" @click="test(storage.id)">{{ phrase('测试连接') }}</button>
              <button type="button" class="button button--ghost" :disabled="busy" @click="edit(storage)">{{ phrase('编辑') }}</button>
              <button type="button" class="button button--ghost button--danger-text" :disabled="busy || settings.schedule.storageId === storage.id" @click="removing = storage; error = ''">{{ phrase('删除') }}</button>
            </div>
          </li>
        </ul>
        <p v-if="settings.schedule.storageId" class="backup-hint">{{ phrase('自动备份使用中的存储，请先在自动备份中更换目标再删除。') }}</p>
        <div v-if="removing" class="backup-options">
          <p>{{ phrase('仅移除连接配置，远程文件仍会保留。') }} {{ removing.name }}</p>
          <div class="backup-option-actions"><button type="button" class="button button--ghost" :disabled="busy" @click="removing = undefined">{{ phrase('取消') }}</button><button type="button" class="button button--danger" :disabled="busy" @click="remove">{{ phrase('确认删除') }}</button></div>
        </div>
        <button v-else type="button" class="button button--primary" :disabled="busy || settings.storages.length >= 8" @click="edit()">{{ phrase('添加远程存储') }}</button>
      </template>
      <form v-else class="backup-options" @submit.prevent="save">
        <fieldset :disabled="busy" class="backup-options">
          <label>{{ phrase('名称') }}<input v-model="editing.name" required maxlength="80" /></label>
          <label>{{ phrase('类型') }}<select v-model="editing.kind" @change="editing.secret = ''; editing.hasSecret = false"><option value="s3">S3</option><option value="webdav">WebDAV</option></select></label>
          <label>{{ phrase('服务地址') }}<input v-model="editing.endpoint" type="url" required placeholder="https://…" maxlength="2048" /></label>
          <template v-if="editing.kind === 's3'">
            <label>Bucket<input v-model="editing.bucket" required maxlength="63" /></label>
            <label>Access Key ID<input v-model="editing.accessKey" required maxlength="256" autocomplete="off" /></label>
          </template>
          <label v-else>{{ phrase('用户名') }}<input v-model="editing.username" required maxlength="256" autocomplete="off" /></label>
          <label>{{ editing.kind === 's3' ? 'Secret Access Key' : phrase('访问密码') }}<input v-model="editing.secret" type="password" :required="!editing.hasSecret" :placeholder="editing.hasSecret ? phrase('留空保持不变') : ''" maxlength="4096" autocomplete="new-password" /></label>
          <label>{{ phrase('备份目录') }}<input v-model="editing.prefix" maxlength="240" placeholder="kpanel" /></label>
          <details v-if="editing.kind === 's3'"><summary>{{ phrase('高级设置') }}</summary><div class="backup-options"><label>Region<input v-model="editing.region" placeholder="us-east-1" maxlength="80" /></label><label class="backup-inline"><input v-model="editing.pathStyle" type="checkbox" />{{ phrase('使用路径寻址（Path style）') }}</label></div></details>
        </fieldset>
        <p class="backup-hint">{{ phrase('建议使用 HTTPS 和独立备份目录。凭据加密保存在当前面板，不写入备份包。') }}</p>
        <footer class="backup-option-actions"><button type="button" class="button button--ghost" :disabled="busy" @click="editing = undefined; error = ''">{{ phrase('返回') }}</button><button type="submit" class="button button--primary" :disabled="busy || !valid">{{ phrase('保存') }}</button></footer>
      </form>
      <p v-if="busy" role="status">{{ phrase('正在处理') }}</p>
      <p v-if="notice" role="status">{{ phrase(notice) }}</p>
      <p v-if="error" class="backup-option-error" role="alert">{{ phrase(error) }}</p>
    </div>
  </ModalDialog>
</template>

<style src="./backup-options.css"></style>
