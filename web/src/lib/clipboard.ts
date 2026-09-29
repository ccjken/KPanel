export async function copyText(value: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value)
      return true
    } catch {
      // HTTP/IP access or browser permissions can make the Clipboard API unavailable.
    }
  }

  const active = document.activeElement instanceof HTMLElement ? document.activeElement : null
  const selection = window.getSelection()
  const ranges = selection ? Array.from({ length: selection.rangeCount }, (_, i) => selection.getRangeAt(i).cloneRange()) : []
  const inputSelection = active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement
    ? [active.selectionStart, active.selectionEnd, active.selectionDirection] as const : null
  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.readOnly = true
  textarea.style.position = 'fixed'
  textarea.style.left = '-10000px'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  try {
    textarea.focus({ preventScroll: true })
    textarea.select()
    return document.execCommand('copy')
  } catch {
    return false
  } finally {
    textarea.remove()
    active?.focus({ preventScroll: true })
    if (inputSelection && inputSelection[0] !== null && inputSelection[1] !== null) {
      (active as HTMLInputElement | HTMLTextAreaElement).setSelectionRange(inputSelection[0], inputSelection[1], inputSelection[2] ?? undefined)
    }
    if (selection) {
      selection.removeAllRanges()
      ranges.forEach(range => selection.addRange(range))
    }
  }
}
