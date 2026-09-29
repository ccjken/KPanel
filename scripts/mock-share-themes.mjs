// Offline visual fixture. Production downloads, persistence and security are
// tested through the Go handler; this preview uses catalog-pinned local bytes.
import { createHash, randomBytes } from 'node:crypto'
import { readFile } from 'node:fs/promises'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
const root = resolve(dirname(fileURLToPath(import.meta.url)), '../share-themes')
const catalog = JSON.parse(await readFile(join(root, 'catalog.json'), 'utf8'))
const prefix = '/api/v1/cluster/share-themes'
const installed = new Map()
let selected = ''
const digest = value => `sha256:${createHash('sha256').update(JSON.stringify(value)).digest('hex')}`
const version = () => digest([selected, [...installed]])
const view = pack => ({ ...pack, resourceVersion: digest([pack, installed.get(pack.id)]), installed: installed.has(pack.id), installedVersion: installed.has(pack.id) ? pack.version : null, fileBase: installed.has(pack.id) ? `${prefix}/${pack.id}/files/${installed.get(pack.id).token}/` : null })
export function activeShareTheme() {
  const pack = catalog.packs.find(pack => pack.id === selected)
  return pack && installed.has(pack.id) ? { id: pack.id, name: pack.name, fileBase: view(pack).fileBase } : undefined
}
export async function mockShareThemes(request, response, url, send, readJSON) {
  if (url.pathname !== prefix && !url.pathname.startsWith(`${prefix}/`)) return false
  const rest = url.pathname.slice(prefix.length).split('/').filter(Boolean)
  const fail = (status, title) => { send(response, status, { status, title }); return true }
  if (!rest.length && request.method === 'GET') {
    send(response, 200, { selected, resourceVersion: version(), source: 'auto', sources: ['auto', 'github', 'mirror'], packs: catalog.packs.map(view) }); return true
  }
  if (rest[0] === 'selection' && rest.length === 1 && request.method === 'PUT') {
    const body = await readJSON(request)
    if (body.expectedResourceVersion !== version()) return fail(409, 'Refresh themes and retry')
    if (body.selected && !installed.has(body.selected)) return fail(404, 'Theme not installed')
    selected = body.selected || ''; send(response, 200, { selected }); return true
  }
  const pack = catalog.packs.find(pack => pack.id === rest[0])
  if (!pack) return fail(404, 'Theme not found')
  if (rest.length === 2 && rest[1] === 'install' && request.method === 'POST') {
    const body = await readJSON(request)
    if (body.expectedResourceVersion !== view(pack).resourceVersion) return fail(409, 'Refresh themes and retry')
    const files = {}
    for (const file of pack.files) {
      const bytes = await readFile(join(root, pack.path, file.path))
      if (bytes.length !== file.size || createHash('sha256').update(bytes).digest('hex') !== file.sha256) return fail(502, 'Theme integrity failed')
      files[file.path] = bytes
    }
    installed.set(pack.id, { token: randomBytes(16).toString('hex'), files })
    send(response, 200, view(pack)); return true
  }
  if (rest.length === 1 && request.method === 'DELETE') {
    const body = await readJSON(request)
    if (body.expectedResourceVersion !== view(pack).resourceVersion) return fail(409, 'Refresh themes and retry')
    installed.delete(pack.id); if (selected === pack.id) selected = ''
    response.writeHead(204); response.end(); return true
  }
  if (rest.length >= 4 && rest[1] === 'files' && request.method === 'GET') {
    const copy = installed.get(pack.id), name = rest.slice(3).join('/'), bytes = copy?.files[name]
    if (copy?.token !== rest[2] || !bytes) return fail(404, 'Theme file not found')
    const base = `http://${request.headers.host}${view(pack).fileBase}`
    const types = { html: 'text/html', css: 'text/css', js: 'text/javascript', json: 'application/json', png: 'image/png', webp: 'image/webp' }
    response.writeHead(200, { 'Content-Type': `${types[name.split('.').pop()] || 'application/octet-stream'}; charset=utf-8`,
      'Cache-Control': 'no-store', 'Access-Control-Allow-Origin': '*', 'Referrer-Policy': 'no-referrer',
      'Content-Security-Policy': `sandbox allow-scripts; default-src 'none'; script-src ${base}; style-src ${base} 'unsafe-inline'; img-src ${base} data:; font-src ${base}; connect-src ${base}; frame-src 'none'; frame-ancestors 'self'; form-action 'none'; base-uri 'none'; object-src 'none'` })
    response.end(bytes); return true
  }
  return fail(405, 'Method not allowed')
}
