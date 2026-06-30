/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Film, FolderPlus, Plus, Trash2, Upload } from 'lucide-react'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import {
  useSeedanceAssetMutations,
  useSeedanceAssets,
  useSeedanceGroups,
} from '../hooks/use-seedance-assets'

export function SeedanceAssetsPage() {
  const { t } = useTranslation()
  const [selectedGroupId, setSelectedGroupId] = useState<number | undefined>()
  const [keyword, setKeyword] = useState('')
  const [groupDialogOpen, setGroupDialogOpen] = useState(false)
  const [assetDialogOpen, setAssetDialogOpen] = useState(false)
  const [groupName, setGroupName] = useState('')
  const [groupDescription, setGroupDescription] = useState('')
  const [assetName, setAssetName] = useState('')
  const [assetType, setAssetType] = useState('image')
  const [assetUrl, setAssetUrl] = useState('')
  const [uploadId, setUploadId] = useState<number | undefined>()

  const groupsQuery = useSeedanceGroups()
  const activeGroupId = useMemo(() => {
    if (selectedGroupId) return selectedGroupId
    const groups = groupsQuery.data ?? []
    const defaultGroup = groups.find((g) => g.is_default) ?? groups[0]
    return defaultGroup?.id
  }, [groupsQuery.data, selectedGroupId])
  const assetsQuery = useSeedanceAssets(activeGroupId, keyword)
  const mutations = useSeedanceAssetMutations()

  const handleCreateGroup = async () => {
    await mutations.createGroup.mutateAsync({
      name: groupName,
      description: groupDescription,
      group_type: 'AIGC',
    })
    setGroupDialogOpen(false)
    setGroupName('')
    setGroupDescription('')
  }

  const handleUpload = async (file: File | undefined) => {
    if (!file) return
    const res = await mutations.uploadFile.mutateAsync(file)
    if (!res.success) {
      return
    }
    setUploadId(res.data.id)
    setAssetUrl(res.data.public_url)
  }

  const handleCreateAsset = async () => {
    if (!activeGroupId) return
    await mutations.createAsset.mutateAsync({
      group_id: activeGroupId,
      name: assetName,
      asset_type: assetType,
      url: uploadId ? undefined : assetUrl,
      upload_id: uploadId,
    })
    setAssetDialogOpen(false)
    setAssetName('')
    setAssetUrl('')
    setUploadId(undefined)
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Seedance Assets')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button variant='outline' onClick={() => setGroupDialogOpen(true)}>
          <FolderPlus className='mr-2 h-4 w-4' />
          {t('New asset group')}
        </Button>
        <Button onClick={() => setAssetDialogOpen(true)} disabled={!activeGroupId}>
          <Plus className='mr-2 h-4 w-4' />
          {t('Add asset')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex h-full min-h-[480px] gap-4'>
          <aside className='border-border w-64 shrink-0 rounded-lg border p-3'>
            <p className='text-muted-foreground mb-2 text-sm font-medium'>
              {t('Asset groups')}
            </p>
            {groupsQuery.isLoading ? (
              <Spinner className='mx-auto' />
            ) : (
              <div className='space-y-1'>
                {(groupsQuery.data ?? []).map((group) => (
                  <button
                    key={group.id}
                    type='button'
                    className={`hover:bg-muted w-full rounded-md px-3 py-2 text-left text-sm ${
                      activeGroupId === group.id ? 'bg-muted font-medium' : ''
                    }`}
                    onClick={() => setSelectedGroupId(group.id)}
                  >
                    <div className='flex items-center justify-between gap-2'>
                      <span className='truncate'>{group.name}</span>
                      {group.is_default ? (
                        <Badge variant='secondary'>{t('Default')}</Badge>
                      ) : null}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </aside>

          <div className='min-w-0 flex-1'>
            <div className='mb-4 flex items-center gap-2'>
              <Input
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                placeholder={t('Search assets by name')}
                className='max-w-sm'
              />
            </div>

            {assetsQuery.isLoading ? (
              <div className='flex justify-center py-16'>
                <Spinner />
              </div>
            ) : (
              <div className='grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3'>
                {(assetsQuery.data?.items ?? []).map((asset) => (
                  <div
                    key={asset.id}
                    className='border-border rounded-lg border p-4'
                  >
                    <div className='mb-2 flex items-start justify-between gap-2'>
                      <div className='min-w-0'>
                        <p className='truncate font-medium'>
                          {asset.name || asset.upstream_id}
                        </p>
                        <p className='text-muted-foreground truncate text-xs'>
                          {asset.asset_type}
                        </p>
                      </div>
                      <Button
                        variant='ghost'
                        size='icon'
                        onClick={() => mutations.removeAsset.mutate(asset.id)}
                      >
                        <Trash2 className='h-4 w-4' />
                      </Button>
                    </div>
                    {asset.public_url || asset.source_url ? (
                      <a
                        href={asset.public_url || asset.source_url}
                        target='_blank'
                        rel='noreferrer'
                        className='text-primary truncate text-xs underline'
                      >
                        {asset.public_url || asset.source_url}
                      </a>
                    ) : (
                      <p className='text-muted-foreground text-xs'>
                        {asset.status || t('Pending')}
                      </p>
                    )}
                    <p className='text-muted-foreground mt-2 text-xs'>
                      {t('Reference')}: seedance_asset://{asset.id}
                    </p>
                  </div>
                ))}
                {(assetsQuery.data?.items?.length ?? 0) === 0 ? (
                  <div className='text-muted-foreground col-span-full flex flex-col items-center justify-center py-16'>
                    <Film className='mb-3 h-10 w-10 opacity-40' />
                    <p>{t('No assets in this group yet')}</p>
                  </div>
                ) : null}
              </div>
            )}
          </div>
        </div>
      </SectionPageLayout.Content>

      <Dialog open={groupDialogOpen} onOpenChange={setGroupDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('New asset group')}</DialogTitle>
          </DialogHeader>
          <div className='space-y-3'>
            <div className='space-y-1'>
              <Label>{t('Name')}</Label>
              <Input value={groupName} onChange={(e) => setGroupName(e.target.value)} />
            </div>
            <div className='space-y-1'>
              <Label>{t('Description')}</Label>
              <Textarea
                value={groupDescription}
                onChange={(e) => setGroupDescription(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={handleCreateGroup}
              disabled={!groupName.trim() || mutations.createGroup.isPending}
            >
              {t('Create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={assetDialogOpen} onOpenChange={setAssetDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('Add asset')}</DialogTitle>
          </DialogHeader>
          <div className='space-y-3'>
            <div className='space-y-1'>
              <Label>{t('Name')}</Label>
              <Input value={assetName} onChange={(e) => setAssetName(e.target.value)} />
            </div>
            <div className='space-y-1'>
              <Label>{t('Asset type')}</Label>
              <Input
                value={assetType}
                onChange={(e) => setAssetType(e.target.value)}
                placeholder='image'
              />
            </div>
            <div className='space-y-1'>
              <Label>{t('Public URL')}</Label>
              <Input
                value={assetUrl}
                onChange={(e) => {
                  setAssetUrl(e.target.value)
                  setUploadId(undefined)
                }}
                placeholder='https://'
              />
            </div>
            <div className='space-y-1'>
              <Label>{t('Upload file')}</Label>
              <Input
                type='file'
                onChange={(e) => void handleUpload(e.target.files?.[0])}
              />
              {uploadId ? (
                <p className='text-muted-foreground text-xs'>
                  {t('Upload ready')}: #{uploadId}
                </p>
              ) : null}
            </div>
          </div>
          <DialogFooter>
            <Button
              onClick={handleCreateAsset}
              disabled={
                !assetType.trim() ||
                (!assetUrl.trim() && !uploadId) ||
                mutations.createAsset.isPending
              }
            >
              <Upload className='mr-2 h-4 w-4' />
              {t('Create')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </SectionPageLayout>
  )
}
