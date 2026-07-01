/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { HugeiconsIcon } from '@hugeicons/react'
import { Add01Icon, FolderAddIcon } from '@hugeicons/core-free-icons'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { AssetGrid } from './asset-grid'
import { CreateAssetDialog, EditAssetDialog } from './asset-dialogs'
import { DeleteAssetDialog } from './delete-asset-dialog'
import { AssetGroupList } from './asset-group-list'
import { CreateGroupDialog, EditGroupDialog } from './group-dialogs'
import {
  useSeedanceAssetMutations,
  useSeedanceAssets,
  useSeedanceGroups,
} from '../hooks/use-seedance-assets'
import type { SeedanceAsset, SeedanceAssetGroup } from '../types'

export function SeedanceAssetsPage() {
  const { t } = useTranslation()
  const [selectedGroupId, setSelectedGroupId] = useState<number | undefined>()
  const [keyword, setKeyword] = useState('')
  const [groupDialogOpen, setGroupDialogOpen] = useState(false)
  const [editGroup, setEditGroup] = useState<SeedanceAssetGroup | null>(null)
  const [assetDialogOpen, setAssetDialogOpen] = useState(false)
  const [editAsset, setEditAsset] = useState<SeedanceAsset | null>(null)
  const [deleteAsset, setDeleteAsset] = useState<SeedanceAsset | null>(null)

  const groupsQuery = useSeedanceGroups()
  const activeGroupId = useMemo(() => {
    if (selectedGroupId) return selectedGroupId
    const groups = groupsQuery.data ?? []
    const defaultGroup = groups.find((g) => g.is_default) ?? groups[0]
    return defaultGroup?.id
  }, [groupsQuery.data, selectedGroupId])
  const assetsQuery = useSeedanceAssets(activeGroupId, keyword)
  const mutations = useSeedanceAssetMutations()

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>{t('Seedance Assets')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Button variant='outline' onClick={() => setGroupDialogOpen(true)}>
            <HugeiconsIcon icon={FolderAddIcon} strokeWidth={2} />
            {t('New asset group')}
          </Button>
          <Button onClick={() => setAssetDialogOpen(true)} disabled={!activeGroupId}>
            <HugeiconsIcon icon={Add01Icon} strokeWidth={2} />
            {t('Add asset')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='flex h-full min-h-[480px] gap-4'>
            <AssetGroupList
              groups={groupsQuery.data ?? []}
              activeGroupId={activeGroupId}
              isLoading={groupsQuery.isLoading}
              onSelect={setSelectedGroupId}
              onEdit={setEditGroup}
            />

            <div className='min-w-0 flex-1'>
              <div className='mb-4 flex items-center gap-2'>
                <Input
                  value={keyword}
                  onChange={(e) => setKeyword(e.target.value)}
                  placeholder={t('Search assets by name')}
                  className='max-w-sm'
                />
              </div>

              <AssetGrid
                assets={assetsQuery.data?.items ?? []}
                isLoading={assetsQuery.isLoading}
                syncingAssetId={
                  mutations.syncAsset.isPending
                    ? mutations.syncAsset.variables
                    : undefined
                }
                onEdit={setEditAsset}
                onDelete={setDeleteAsset}
                onSync={(asset) => mutations.syncAsset.mutate(asset.id)}
              />
            </div>
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <CreateGroupDialog
        open={groupDialogOpen}
        onOpenChange={setGroupDialogOpen}
        isPending={mutations.createGroup.isPending}
        onSubmit={async (values) => {
          await mutations.createGroup.mutateAsync({
            name: values.name,
            description: values.description,
            group_type: 'AIGC',
          })
          setGroupDialogOpen(false)
        }}
      />

      <EditGroupDialog
        open={editGroup != null}
        onOpenChange={(open) => {
          if (!open) setEditGroup(null)
        }}
        initialName={editGroup?.name ?? ''}
        initialDescription={editGroup?.description ?? ''}
        isPending={mutations.updateGroup.isPending}
        onSubmit={async (values) => {
          if (!editGroup) return
          await mutations.updateGroup.mutateAsync({
            id: editGroup.id,
            name: values.name,
            description: values.description,
          })
          setEditGroup(null)
        }}
      />

      <CreateAssetDialog
        open={assetDialogOpen}
        onOpenChange={setAssetDialogOpen}
        isPending={mutations.createAsset.isPending}
        onUpload={async (file) => {
          const res = await mutations.uploadFile.mutateAsync(file)
          if (!res.success) {
            toast.error(res.message)
            return {}
          }
          return { uploadId: res.data.id, publicUrl: res.data.public_url }
        }}
        onSubmit={async (values) => {
          if (!activeGroupId) return
          await mutations.createAsset.mutateAsync({
            group_id: activeGroupId,
            name: values.name,
            asset_type: values.assetType,
            url: values.url,
            upload_id: values.uploadId,
          })
          setAssetDialogOpen(false)
        }}
      />

      <EditAssetDialog
        open={editAsset != null}
        onOpenChange={(open) => {
          if (!open) setEditAsset(null)
        }}
        initialName={editAsset?.name ?? ''}
        isPending={mutations.updateAsset.isPending}
        onSubmit={async (name) => {
          if (!editAsset) return
          await mutations.updateAsset.mutateAsync({ id: editAsset.id, name })
          setEditAsset(null)
        }}
      />

      <DeleteAssetDialog
        asset={deleteAsset}
        open={deleteAsset != null}
        onOpenChange={(open) => {
          if (!open) setDeleteAsset(null)
        }}
        isPending={mutations.removeAsset.isPending}
        onConfirm={() => {
          if (!deleteAsset) return
          mutations.removeAsset.mutate(deleteAsset.id, {
            onSuccess: () => setDeleteAsset(null),
          })
        }}
      />
    </>
  )
}
