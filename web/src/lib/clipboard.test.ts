// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { copyText } from './clipboard'

afterEach(() => {
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
  document.body.replaceChildren()
  window.getSelection()?.removeAllRanges()
})

describe('copyText', () => {
  const value = '**完整回答**\n中文与 <code> 保持原样\n  echo hello\n'

  it('uses the async clipboard without changing focus or adding an input', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const fallback = vi.fn()
    Object.defineProperty(document, 'execCommand', { value: fallback, configurable: true })
    expect(await copyText(value)).toBe(true)
    expect(writeText).toHaveBeenCalledWith(value)
    expect(fallback).not.toHaveBeenCalled()
    expect(document.querySelector('textarea')).toBeNull()
  })

  it.each(['missing', 'denied'])('copies the full selection when the API is %s and restores composer focus', async mode => {
    vi.stubGlobal('navigator', mode === 'missing' ? {} : { clipboard: { writeText: vi.fn().mockRejectedValue(new Error('denied')) } })
    const composer = document.createElement('textarea')
    composer.value = '保留输入草稿'
    document.body.appendChild(composer)
    composer.focus()
    composer.setSelectionRange(1, 4, 'backward')
    const fallback = vi.fn(() => {
      const selected = document.activeElement as HTMLTextAreaElement
      expect(selected.value).toBe(value)
      expect(selected.selectionStart).toBe(0)
      expect(selected.selectionEnd).toBe(value.length)
      return true
    })
    Object.defineProperty(document, 'execCommand', { value: fallback, configurable: true })
    expect(await copyText(value)).toBe(true)
    expect(fallback).toHaveBeenCalledWith('copy')
    expect(document.activeElement).toBe(composer)
    expect(composer.value).toBe('保留输入草稿')
    expect([composer.selectionStart, composer.selectionEnd, composer.selectionDirection]).toEqual([1, 4, 'backward'])
    expect(document.querySelectorAll('textarea')).toHaveLength(1)
  })

  it.each(['false', 'throws'])('reports fallback failure (%s) and removes temporary elements', async mode => {
    vi.stubGlobal('navigator', {})
    Object.defineProperty(document, 'execCommand', { configurable: true, value: vi.fn(() => {
      if (mode === 'throws') throw new Error('blocked')
      return false
    }) })
    const button = document.createElement('button')
    document.body.appendChild(button)
    button.focus()
    expect(await copyText(value)).toBe(false)
    expect(document.activeElement).toBe(button)
    expect(document.querySelector('textarea')).toBeNull()
  })

  it('restores the reader selection after a fallback attempt', async () => {
    vi.stubGlobal('navigator', {})
    Object.defineProperty(document, 'execCommand', { configurable: true, value: () => true })
    const paragraph = document.createElement('p')
    paragraph.textContent = value
    document.body.appendChild(paragraph)
    const range = document.createRange()
    range.selectNodeContents(paragraph)
    window.getSelection()!.addRange(range)
    await copyText('a different code block')
    expect(window.getSelection()!.toString()).toBe(value)
  })
})
