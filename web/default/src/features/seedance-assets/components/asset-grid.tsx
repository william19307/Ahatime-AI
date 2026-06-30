/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useTranslation } from 'react-i18next'
import { HugeiconsIcon } from '@hugeicons/react'
import {
  Delete02Icon,
  Edit02Icon,
  RefreshIcon,
  Video01Icon,
} from '@hugeicons/core-free-icons'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import {
  isSeedanceAudioAsset,
  isSeedanceImageAsset,
  isSeedanceVideoAsset,
  resolveSeedanceAssetPreviewUrl,
} from '../lib/preview'
import type { SeedanceAsset } from '../types'

type AssetPreviewProps = {
  asset: SeedanceAsset
}

function AssetPreview(props: AssetPreviewProps) {
  const url = resolveSeedanceAssetPreviewUrl(props.asset)
  if (!url) return null

  if (isSeedanceImageAsset(props.asset)) {
    return (
      <img
        src={url}
        alt={props.asset.name || props.asset.upstream_id}
        className='bg-muted mb-3 aspect-video w-full rounded-md object-cover'
      />
    )
  }
  if (isSeedanceVideoAsset(props.asset)) {
    return (
      <video
        src={url}
        controls
        className='bg-muted mb-3 aspect-video w-full rounded-md object-cover'
      />
    )
  }
  if (isSeedanceAudioAsset(props.asset)) {
    return <audio src={url} controls className='mb-3 w-full' />
  }
  return null
}

type AssetCardProps = {
  asset: SeedanceAsset
  onEdit: (asset: SeedanceAsset) => void
  onDelete: (asset: SeedanceAsset) => void
  onSync: (asset: SeedanceAsset) => void
  isSyncing: boolean
}

export function AssetCard(props: AssetCardProps) {
  const { t } = useTranslation()
  const previewUrl = resolveSeedanceAssetPreviewUrl(props.asset)

  return (
    <Card className='overflow-hidden'>
      <CardHeader className='flex flex-row items-start justify-between gap-2 space-y-0 pb-2'>
        <div className='min-w-0'>
          <CardTitle className='truncate text-base'>
            {props.asset.name || props.asset.upstream_id}
          </CardTitle>
          <p className='text-muted-foreground truncate text-xs'>
            {props.asset.asset_type}
            {props.asset.status ? ` · ${props.asset.status}` : ''}
          </p>
        </div>
        <div className='flex shrink-0 items-center gap-1'>
          <Button
            variant='ghost'
            size='icon'
            aria-label={t('Sync from upstream')}
            onClick={() => props.onSync(props.asset)}
            disabled={props.isSyncing}
          >
            <HugeiconsIcon icon={RefreshIcon} strokeWidth={2} />
          </Button>
          <Button
            variant='ghost'
            size='icon'
            aria-label={t('Rename asset')}
            onClick={() => props.onEdit(props.asset)}
          >
            <HugeiconsIcon icon={Edit02Icon} strokeWidth={2} />
          </Button>
          <Button
            variant='ghost'
            size='icon'
            aria-label={t('Delete asset')}
            onClick={() => props.onDelete(props.asset)}
          >
            <HugeiconsIcon icon={Delete02Icon} strokeWidth={2} />
          </Button>
        </div>
      </CardHeader>
      <CardContent className='space-y-2'>
        <AssetPreview asset={props.asset} />
        {previewUrl ? (
          <a
            href={previewUrl}
            target='_blank'
            rel='noreferrer'
            className='text-primary block truncate text-xs underline'
          >
            {previewUrl}
          </a>
        ) : (
          <p className='text-muted-foreground text-xs'>{t('Pending')}</p>
        )}
        <p className='text-muted-foreground text-xs'>
          {t('Reference')}: seedance_asset://{props.asset.id}
        </p>
      </CardContent>
    </Card>
  )
}

type AssetGridProps = {
  assets: SeedanceAsset[]
  isLoading: boolean
  syncingAssetId?: number
  onEdit: (asset: SeedanceAsset) => void
  onDelete: (asset: SeedanceAsset) => void
  onSync: (asset: SeedanceAsset) => void
}

export function AssetGrid(props: AssetGridProps) {
  const { t } = useTranslation()

  if (props.isLoading) {
    return (
      <div className='flex justify-center py-16'>
        <Spinner className='mx-auto' />
      </div>
    )
  }

  if (props.assets.length === 0) {
    return (
      <div className='text-muted-foreground flex flex-col items-center justify-center py-16'>
        <HugeiconsIcon icon={Video01Icon} className='mb-3 size-10 opacity-40' />
        <p>{t('No assets in this group yet')}</p>
      </div>
    )
  }

  return (
    <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3'>
      {props.assets.map((asset) => (
        <AssetCard
          key={asset.id}
          asset={asset}
          onEdit={props.onEdit}
          onDelete={props.onDelete}
          onSync={props.onSync}
          isSyncing={props.syncingAssetId === asset.id}
        />
      ))}
    </div>
  )
}
