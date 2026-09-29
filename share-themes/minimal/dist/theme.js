// Readable reference renderer. All user content is inserted as text, never HTML.
let state
let query = ''
const app = document.getElementById('app')
const zh = { fleet: '公开集群', total: '全部机器', online: '在线', attention: '需关注', remaining: '剩余价值（估算）', coverage: '台已计入', excluded: '台资料不足', search: '搜索名称、地区或系统', empty: '没有匹配的机器', memory: '内存', disk: '磁盘', uptime: '运行时间', monthly: '月流量', cumulative: '累计流量', expiry: '到期', price: '价格', offline: '离线', pending: '等待数据', degraded: '需关注', updated: '数据生成于', valueHint: '按已公开价格及到期日估算，分币种展示。', noValue: '资料不足，暂无法估算' }
const tw = { ...zh, fleet: '公開集群', total: '全部機器', online: '在線', attention: '需關注', remaining: '剩餘價值（估算）', coverage: '台已計入', excluded: '台資料不足', search: '搜尋名稱、地區或系統', empty: '沒有符合的機器', memory: '記憶體', disk: '磁碟', uptime: '運行時間', monthly: '月流量', cumulative: '累計流量', expiry: '到期', price: '價格', offline: '離線', pending: '等待資料', degraded: '需關注', updated: '資料產生於', valueHint: '按已公開價格及到期日估算，分幣種展示。', noValue: '資料不足，暫時無法估算' }
const en = { fleet: 'PUBLIC FLEET', total: 'Servers', online: 'Online', attention: 'Attention', remaining: 'Remaining value (estimate)', coverage: 'included', excluded: 'incomplete', search: 'Search name, location or OS', empty: 'No matching servers', memory: 'Memory', disk: 'Disk', uptime: 'Uptime', monthly: 'Monthly traffic', cumulative: 'Total traffic', expiry: 'Expires', price: 'Price', offline: 'Offline', pending: 'Awaiting data', degraded: 'Attention', updated: 'Updated', valueHint: 'Estimated from public price and expiry, grouped by currency.', noValue: 'Not enough details to estimate' }
function el(tag, text, className) {
  const node = document.createElement(tag)
  if (text !== undefined) node.textContent = String(text)
  if (className) node.className = className
  return node
}
function metric(label, value, className = '') {
  const node = el('div', undefined, `metric ${className}`)
  node.append(el('small', label), el('strong', value)); return node
}
function renderHosts(container, words) {
  container.replaceChildren()
  const hosts = state.data.hosts.filter(host => `${host.name} ${host.location} ${host.os}`.toLowerCase().includes(query.toLowerCase()))
  if (!hosts.length) container.append(el('p', words.empty, 'empty'))
  for (const host of hosts) {
    const card = el('article', undefined, 'host')
    const top = el('header')
    const identity = el('div')
    identity.append(el('h2', host.name), el('p', [host.location, host.os].filter(Boolean).join(' · ')))
    top.append(identity, el('span', words[host.state] || words.pending, `status ${host.state}`))
    const resources = el('div', undefined, 'resources')
    resources.append(metric('CPU', host.cpu), metric(words.memory, host.memory), metric(words.disk, host.disk))
    const traffic = el('div', undefined, `traffic ${host.traffic.monthly ? 'monthly' : ''} ${host.traffic.tone}`)
    traffic.append(el('strong', `${host.traffic.monthly ? words.monthly : words.cumulative}${host.traffic.percent ? ` · ${host.traffic.percent}` : ''}`), el('span', `↓ ${host.traffic.received}   ↑ ${host.traffic.sent}`))
    const details = el('dl')
    for (const [label, value] of [[words.uptime, host.uptime], [words.expiry, host.expiresOn], [words.price, host.price], [words.remaining, host.remaining]]) {
      if (value) details.append(el('dt', label), el('dd', value))
    }
    card.append(top, resources, traffic, details); container.append(card)
  }
}
function render() {
  const words = state.locale === 'en-US' ? en : state.locale === 'zh-TW' ? tw : zh
  const focused = document.activeElement?.id === 'search'
  const selection = focused ? [document.activeElement.selectionStart, document.activeElement.selectionEnd] : null
  const expanded = Boolean(document.querySelector('details')?.open)
  document.documentElement.lang = state.locale
  document.documentElement.dataset.mode = state.mode === 'light' ? 'light' : 'dark'
  document.title = state.data.title
  const hero = el('header', undefined, 'hero')
  hero.append(el('span', words.fleet, 'eyebrow'), el('h1', state.data.title), el('p', state.data.description))
  const stats = el('section', undefined, 'stats')
  stats.append(metric(words.total, state.data.total), metric(words.online, state.data.online, 'online'), metric(words.attention, state.data.attention))
  const value = el('details', undefined, 'value'); value.open = expanded
  const summary = el('summary')
  summary.append(el('small', words.remaining), el('strong', state.data.value.groups[0]?.text || '—'))
  if (state.data.value.groups.length > 1) summary.append(el('small', `+${state.data.value.groups.length - 1}`))
  value.append(summary, el('p', words.valueHint))
  for (const group of state.data.value.groups) value.append(el('p', `${group.currency} · ${group.text}`))
  value.append(el('p', `${state.data.value.included} / ${state.data.total} ${words.coverage} · ${state.data.value.excluded} ${words.excluded}`))
  if (!state.data.value.groups.length) value.append(el('p', words.noValue))
  stats.append(value)
  const label = el('label', words.search, 'search')
  const input = el('input'); input.type = 'search'; input.id = 'search'; input.value = query; input.placeholder = words.search
  label.append(input)
  const hosts = el('section', undefined, 'hosts')
  input.addEventListener('input', () => { query = input.value; renderHosts(hosts, words) })
  renderHosts(hosts, words)
  const date = new Date(state.data.generatedAt)
  const footer = el('footer', `${words.updated} ${Number.isFinite(date.getTime()) ? date.toLocaleString(state.locale) : '—'} · KPanel`)
  app.replaceChildren(hero, stats, label, hosts, footer)
  if (focused) { input.focus(); input.setSelectionRange(...selection) }
}
addEventListener('message', event => {
  if (event.source !== parent || event.data?.source !== 'kpanel-share' || event.data?.type !== 'snapshot' || event.data?.schema !== 1 || !Array.isArray(event.data?.data?.hosts)) return
  state = event.data; render()
})
parent.postMessage({ source: 'kpanel-share-theme', type: 'ready' }, '*')
