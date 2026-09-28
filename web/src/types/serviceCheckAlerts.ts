export interface CheckSubscription { hostId: string; checkId: string; revision: string }
export interface CheckAlertSettings { enabled: boolean; repeat: boolean; subscriptions: CheckSubscription[] }
export interface CheckAlertSnapshot extends CheckAlertSettings {
  resourceVersion: string
  pending: number
  lastError?: string
  channelReady: boolean
  hosts: Array<{
    id: string; isLocal: boolean; name: string; state: string
    checks: null | { available: boolean; intervalSeconds: number; items: Array<{
      id: string; revision: string; name: string; target: string; kind: string; state: string; checkedAt: string
    }> }
  }>
}
