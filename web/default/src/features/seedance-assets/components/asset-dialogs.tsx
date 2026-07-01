/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { HugeiconsIcon } from '@hugeicons/react'
import { Upload01Icon } from '@hugeicons/core-free-icons'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { SEEDANCE_ASSET_TYPES } from '../constants'
import {
  inferSeedanceAssetTypeFromFile,
  validateSeedanceImageFileDimensions,
} from '../lib/asset-type'

type CreateAssetDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  isPending: boolean
  onUpload: (file: File) => Promise<{ uploadId?: number; publicUrl?: string }>
  onSubmit: (values: {
    name: string
    assetType: string
    url?: string
    uploadId?: number
  }) => Promise<void>
}

export function CreateAssetDialog(props: CreateAssetDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [assetType, setAssetType] = useState<string>('Image')
  const [assetUrl, setAssetUrl] = useState('')
  const [uploadId, setUploadId] = useState<number | undefined>()

  useEffect(() => {
    if (!props.open) {
      setName('')
      setAssetType('Image')
      setAssetUrl('')
      setUploadId(undefined)
    }
  }, [props.open])

  const handleUpload = async (file: File | undefined) => {
    if (!file) return
    const dimensionError = await validateSeedanceImageFileDimensions(file)
    if (dimensionError) {
      toast.error(dimensionError)
      return
    }
    const inferred = inferSeedanceAssetTypeFromFile(file)
    if (inferred) setAssetType(inferred)
    const result = await props.onUpload(file)
    if (result.uploadId) setUploadId(result.uploadId)
    if (result.publicUrl) setAssetUrl(result.publicUrl)
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('Add asset')}</DialogTitle>
        </DialogHeader>
        <div className='space-y-3'>
          <div className='space-y-1'>
            <Label>{t('Name')}</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className='space-y-1'>
            <Label>{t('Asset type')}</Label>
            <Select
              value={assetType}
              onValueChange={(value) => setAssetType(value ?? 'Image')}
            >
              <SelectTrigger>
                <SelectValue placeholder={t('Asset type')} />
              </SelectTrigger>
              <SelectContent>
                {SEEDANCE_ASSET_TYPES.map((type) => (
                  <SelectItem key={type} value={type}>
                    {type}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input
              value={assetType}
              onChange={(e) => setAssetType(e.target.value)}
              placeholder={t('Custom asset type')}
              className='mt-2'
            />
          </div>
          <p className='text-muted-foreground text-xs'>
            {t(
              'Seedance image requirements: each side 300-6000 px, aspect ratio 0.4-2.5, max 30 MB.',
            )}
          </p>
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
            onClick={() =>
              void props.onSubmit({
                name,
                assetType,
                url: uploadId ? undefined : assetUrl,
                uploadId,
              })
            }
            disabled={
              !assetType.trim() ||
              (!assetUrl.trim() && !uploadId) ||
              props.isPending
            }
          >
            <HugeiconsIcon icon={Upload01Icon} strokeWidth={2} />
            {t('Create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type EditAssetDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialName: string
  isPending: boolean
  onSubmit: (name: string) => Promise<void>
}

export function EditAssetDialog(props: EditAssetDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState(props.initialName)

  useEffect(() => {
    if (props.open) setName(props.initialName)
  }, [props.open, props.initialName])

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('Rename asset')}</DialogTitle>
        </DialogHeader>
        <div className='space-y-1'>
          <Label>{t('Name')}</Label>
          <Input value={name} onChange={(e) => setName(e.target.value)} />
        </div>
        <DialogFooter>
          <Button
            onClick={() => void props.onSubmit(name)}
            disabled={!name.trim() || props.isPending}
          >
            {t('Save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
