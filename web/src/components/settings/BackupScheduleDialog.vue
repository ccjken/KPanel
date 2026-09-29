<script setup lang="ts">
import { computed, ref } from 'vue'
import ModalDialog from '@/components/common/ModalDialog.vue'
import { backups, backupMessage, type BackupSettings, type BackupModule } from '@/lib/backup'
import { phraseCatalogVersion, translatePhrase } from '@/i18n/phrase'
import { getLocale } from '@/i18n'
const props = defineProps<{ settings: BackupSettings; pending: boolean }>()
const emit = defineEmits<{ close: []; saved: [settings: BackupSettings]; started: [] }>()
function phrase(value: string) { phraseCatalogVersion.value; return translatePhrase(value) }
const plan = ref({ ...props.settings.schedule, modules: [...props.settings.schedule.modules], password: '' })
const clock = ref(`${String(plan.value.hour).padStart(2, '0')}:${String(plan.value.minute).padStart(2, '0')}`)
const busy = ref(false)
const error = ref('')
const notice = ref('')
const confirmation = ref('')
const modules: Array<[BackupModule, string]> = [['panel', '面板数据'], ['apps', '应用数据'], ['web', '网站数据'], ['docker', 'Docker 数据']]
const weekdays = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六']
const nextRun = computed(() => props.settings.schedule.nextRun ? new Intl.DateTimeFormat(getLocale(), { timeZone: props.settings.schedule.timezone, dateStyle: 'medium', timeStyle: 'short' }).format(new Date(props.settings.schedule.nextRun)) : '')
const valid = computed(() => plan.value.modules.length > 0 && !!clock.value && !!plan.value.timezone && (plan.value.password ? new TextEncoder().encode(plan.value.password).length >= 10 && new TextEncoder().encode(plan.value.password).length <= 256 && plan.value.password === confirmation.value : !plan.value.enabled || plan.value.hasPassword))
async function save() {
  if (!valid.value || busy.value) return
  busy.value = true; error.value = ''; notice.value = ''
  const [hour, minute] = clock.value.split(':').map(Number)
  try {
    const settings = await backups.saveSchedule(props.settings.revision, { ...plan.value, hour: hour!, minute: minute! })
    emit('saved', settings); plan.value = { ...settings.schedule, modules: [...settings.schedule.modules], password: '' }; confirmation.value = ''; notice.value = '自动备份设置已保存。'
  } catch (reason) { error.value = backupMessage(reason) }
  finally { busy.value = false }
}
async function run() {
  if (busy.value || props.pending) return
  busy.value = true; error.value = ''; notice.value = ''
  try { await backups.runSchedule(); emit('started'); notice.value = '任务已受理，可在下方查看进度。' }
  catch (reason) { error.value = backupMessage(reason) }
  finally { busy.value = false }
}
</script>

<template>
  <ModalDialog :open="true" :title="phrase('自动备份')" size="medium" :close-disabled="busy" @close="emit('close')">
    <form class="backup-options" @submit.prevent="save">
      <fieldset :disabled="busy" class="backup-options">
        <label class="backup-inline"><input v-model="plan.enabled" type="checkbox" />{{ phrase('启用定时备份') }}</label>
        <div class="backup-option-grid">
          <label>{{ phrase('频率') }}<select v-model="plan.frequency"><option value="daily">{{ phrase('每天') }}</option><option value="weekly">{{ phrase('每周') }}</option><option value="monthly">{{ phrase('每月') }}</option></select></label>
          <label>{{ phrase('执行时间') }}<input v-model="clock" type="time" required /></label>
          <label v-if="plan.frequency === 'weekly'">{{ phrase('执行日期') }}<select v-model.number="plan.weekday"><option v-for="(day, i) in weekdays" :key="day" :value="i">{{ phrase(day) }}</option></select></label>
          <label v-if="plan.frequency === 'monthly'">{{ phrase('每月几号') }}<input v-model.number="plan.day" type="number" min="1" max="31" required /></label>
        </div>
        <label>{{ phrase('时区') }}<input v-model="plan.timezone" required maxlength="80" placeholder="Asia/Shanghai" /></label>
        <fieldset><legend>{{ phrase('选择内容') }}</legend><div class="backup-option-grid"><label v-for="[id, name] in modules" :key="id" class="backup-inline"><input v-model="plan.modules" type="checkbox" :value="id" />{{ phrase(name) }}</label></div></fieldset>
        <label>{{ phrase('保存到') }}<select v-model="plan.storageId"><option value="">{{ phrase('仅本机') }}</option><option v-for="storage in settings.storages" :key="storage.id" :value="storage.id">{{ storage.name }}</option></select></label>
        <label>{{ phrase('保留最近成功份数') }}<input v-model.number="plan.keep" type="number" min="1" max="20" required /></label>
        <label>{{ phrase('备份密码') }}<input v-model="plan.password" type="password" autocomplete="new-password" :required="plan.enabled && !plan.hasPassword" :placeholder="plan.hasPassword ? phrase('留空保持不变') : ''" maxlength="256" /></label>
        <label v-if="plan.password">{{ phrase('再次输入密码') }}<input v-model="confirmation" type="password" autocomplete="new-password" required maxlength="256" /></label>
      </fieldset>
      <p class="backup-hint">{{ phrase('密码至少 10 字节，建议使用 10 个以上英文字符；忘记密码将无法恢复备份。') }}</p>
      <p class="backup-hint">{{ phrase('应用、网站和 Docker 备份会短暂停止相关容器，完成后恢复原运行状态。') }}</p>
      <p class="backup-hint">{{ phrase('仅清理本计划在同一目标中创建的成功备份。远程传输失败会保留本地文件，可手动重试。') }}</p>
      <details><summary>{{ phrase('运行说明') }}</summary><p class="backup-hint">{{ phrase('面板需保持运行。错过超过 5 分钟、无效月日或夏令时跳过的时间不补跑；恢复面板数据后自动备份暂停。密码与计划加密保存在本机，不随备份迁移。') }}</p></details>
      <p v-if="settings.schedule.enabled && nextRun">{{ phrase('下次执行：') }} {{ nextRun }} · {{ settings.schedule.timezone }}</p>
      <p v-if="settings.schedule.lastError" class="backup-option-error">{{ phrase(settings.schedule.lastError === 'schedule_missed' ? '上次计划已错过，请检查面板运行状态。' : '上次自动备份未完成，请查看备份记录。') }}</p>
      <p v-if="notice" role="status">{{ phrase(notice) }}</p><p v-if="error" role="alert" class="backup-option-error">{{ phrase(error) }}</p>
      <footer class="backup-option-actions"><button type="button" class="button button--secondary" :disabled="busy || pending || !settings.schedule.hasPassword" @click="run">{{ phrase('立即执行已保存计划') }}</button><button type="submit" class="button button--primary" :disabled="busy || !valid">{{ phrase('保存') }}</button></footer>
    </form>
  </ModalDialog>
</template>

<style src="./backup-options.css"></style>
