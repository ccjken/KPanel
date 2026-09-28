import type { PhraseCatalog } from '@/i18n/phrase'

export default [
  [
    "服务异常通知",
    "Service alerts"
  ],
  [
    "复用集群通知渠道，按主机选择要关注的检测项。",
    "Use the cluster notification channel and select checks per host."
  ],
  [
    "正在读取检测项",
    "Loading checks"
  ],
  [
    "读取失败",
    "Could not load settings"
  ],
  [
    "无法读取服务通知设置。",
    "Could not load service alert settings."
  ],
  [
    "设置已被修改，请重新读取后再保存。",
    "Settings have changed. Reload before saving."
  ],
  [
    "保存失败，当前选择尚未生效。",
    "Save failed. Your changes have not been applied."
  ],
  [
    "连续 3 次失败告警，连续 2 次成功恢复。沿用 5 分钟检测周期，通常需 10–15 分钟确认异常。",
    "Alert after 3 failed checks; recover after 2 successful checks. The existing 5-minute interval usually confirms failures in 10–15 minutes."
  ],
  [
    "请先在集群通知中启用并配置可用渠道。可以先保存订阅。",
    "Enable and configure a channel in cluster notifications. You can save subscriptions first."
  ],
  [
    "启用服务异常通知",
    "Enable service alerts"
  ],
  [
    "异常持续时每 6 小时提醒一次",
    "Repeat every 6 hours while unavailable"
  ],
  [
    "主机失联时抑制服务通知；检测数据中断单独提醒。默认不订阅任何检测项。",
    "Service alerts are suppressed while the host is offline. Missing check data is reported separately. No checks are selected by default."
  ],
  [
    "暂无检测摘要，请确认节点版本与连接状态。",
    "No check summary. Check the node version and connection."
  ],
  [
    "检测暂不可用，已有订阅保留。",
    "Checks are unavailable. Existing subscriptions are retained."
  ],
  [
    "正常",
    "Healthy"
  ],
  [
    "异常",
    "Unavailable"
  ],
  [
    "等待检测",
    "Awaiting check"
  ],
  [
    "暂无可订阅主机。",
    "No hosts available for subscription."
  ],
  [
    "检测已修改、删除或暂不可用，请核对订阅。",
    "The check was changed, removed or is unavailable. Review this subscription."
  ],
  [
    "移除订阅",
    "Remove subscription"
  ],
  [
    "待发送通知",
    "Pending notifications"
  ],
  [
    "通知发送或存储异常，请检查通知渠道和磁盘状态。",
    "Notification delivery or storage failed. Check the channel and disk."
  ],
  [
    "服务通知设置已保存。",
    "Service alert settings saved."
  ],
  [
    "重新读取",
    "Reload"
  ],
  [
    "关闭",
    "Close"
  ],
  [
    "正在保存…",
    "Saving…"
  ],
  [
    "保存设置",
    "Save settings"
  ]
] satisfies PhraseCatalog
