/** Convert bounded raster uploads to one self-contained 128px PNG. No remote fetch. */
export async function prepareSiteIcon(file: File): Promise<string> {
  if (!['image/png', 'image/jpeg', 'image/webp'].includes(file.type) || file.size > 2 * 1024 * 1024 || !file.size) {
    throw new Error('Invalid icon')
  }
  const url = URL.createObjectURL(file)
  try {
    const image = new Image()
    await new Promise<void>((resolve, reject) => {
      image.onload = () => resolve()
      image.onerror = () => reject(new Error('Invalid image'))
      image.src = url
    })
    if (!image.naturalWidth || !image.naturalHeight || image.naturalWidth > 4096 || image.naturalHeight > 4096) throw new Error('Invalid dimensions')
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = 128
    const context = canvas.getContext('2d')
    if (!context) throw new Error('Canvas unavailable')
    const scale = Math.min(128 / image.naturalWidth, 128 / image.naturalHeight)
    const width = image.naturalWidth * scale
    const height = image.naturalHeight * scale
    context.drawImage(image, (128 - width) / 2, (128 - height) / 2, width, height)
    const result = canvas.toDataURL('image/png')
    if (!result.startsWith('data:image/png;base64,') || result.length > 87406) throw new Error('Icon too large')
    return result
  } finally { URL.revokeObjectURL(url) }
}
