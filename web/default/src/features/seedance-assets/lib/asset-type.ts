/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import type { SeedanceAssetType } from '../constants'

export const SEEDANCE_IMAGE_MIN_PX = 300
export const SEEDANCE_IMAGE_MAX_PX = 6000
export const SEEDANCE_IMAGE_MIN_ASPECT = 0.4
export const SEEDANCE_IMAGE_MAX_ASPECT = 2.5

export function inferSeedanceAssetTypeFromFile(
  file: File,
): SeedanceAssetType | undefined {
  const mime = file.type.toLowerCase()
  if (mime.startsWith('image/')) return 'Image'
  if (mime.startsWith('video/')) return 'Video'
  if (mime.startsWith('audio/')) return 'Audio'

  const name = file.name.toLowerCase()
  if (/\.(png|jpe?g|gif|webp)$/.test(name)) return 'Image'
  if (/\.(mp4|webm|mov)$/.test(name)) return 'Video'
  if (/\.(mp3|wav|ogg|m4a)$/.test(name)) return 'Audio'

  return undefined
}

export function validateSeedanceImageFileDimensions(
  file: File,
): Promise<string | undefined> {
  if (!file.type.startsWith('image/') && !/\.(png|jpe?g|gif|webp)$/i.test(file.name)) {
    return Promise.resolve(undefined)
  }

  return new Promise((resolve) => {
    const url = URL.createObjectURL(file)
    const img = new Image()
    img.onload = () => {
      URL.revokeObjectURL(url)
      const { width, height } = img
      if (width < SEEDANCE_IMAGE_MIN_PX || height < SEEDANCE_IMAGE_MIN_PX) {
        resolve(
          `Image is too small (${width}x${height}). Each side must be ${SEEDANCE_IMAGE_MIN_PX}-${SEEDANCE_IMAGE_MAX_PX}px.`,
        )
        return
      }
      if (width > SEEDANCE_IMAGE_MAX_PX || height > SEEDANCE_IMAGE_MAX_PX) {
        resolve(
          `Image is too large (${width}x${height}). Each side must be ${SEEDANCE_IMAGE_MIN_PX}-${SEEDANCE_IMAGE_MAX_PX}px.`,
        )
        return
      }
      const aspect = width / height
      if (aspect < SEEDANCE_IMAGE_MIN_ASPECT || aspect > SEEDANCE_IMAGE_MAX_ASPECT) {
        resolve(
          `Image aspect ratio ${aspect.toFixed(2)} is out of range (${SEEDANCE_IMAGE_MIN_ASPECT}-${SEEDANCE_IMAGE_MAX_ASPECT}).`,
        )
        return
      }
      resolve(undefined)
    }
    img.onerror = () => {
      URL.revokeObjectURL(url)
      resolve(undefined)
    }
    img.src = url
  })
}
