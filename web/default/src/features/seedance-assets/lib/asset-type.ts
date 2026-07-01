/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import type { SeedanceAssetType } from '../constants'

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
