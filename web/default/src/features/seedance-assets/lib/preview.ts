/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import type { SeedanceAsset } from '../types'

export function resolveSeedanceAssetPreviewUrl(asset: SeedanceAsset): string {
  return asset.public_url || asset.source_url || ''
}

export function isSeedanceImageAsset(asset: SeedanceAsset): boolean {
  const type = asset.asset_type.toLowerCase()
  const url = resolveSeedanceAssetPreviewUrl(asset).toLowerCase()
  return type.includes('image') || /\.(png|jpe?g|gif|webp)(\?|$)/.test(url)
}

export function isSeedanceVideoAsset(asset: SeedanceAsset): boolean {
  const type = asset.asset_type.toLowerCase()
  const url = resolveSeedanceAssetPreviewUrl(asset).toLowerCase()
  return type.includes('video') || /\.(mp4|webm|mov)(\?|$)/.test(url)
}

export function isSeedanceAudioAsset(asset: SeedanceAsset): boolean {
  const type = asset.asset_type.toLowerCase()
  const url = resolveSeedanceAssetPreviewUrl(asset).toLowerCase()
  return type.includes('audio') || /\.(mp3|wav|ogg|m4a)(\?|$)/.test(url)
}
