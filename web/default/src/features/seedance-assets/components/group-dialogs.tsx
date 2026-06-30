/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
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

type CreateGroupDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  isPending: boolean
  onSubmit: (values: { name: string; description: string }) => Promise<void>
}

export function CreateGroupDialog(props: CreateGroupDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')

  useEffect(() => {
    if (!props.open) {
      setName('')
      setDescription('')
    }
  }, [props.open])

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('New asset group')}</DialogTitle>
        </DialogHeader>
        <div className='space-y-3'>
          <div className='space-y-1'>
            <Label>{t('Name')}</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className='space-y-1'>
            <Label>{t('Description')}</Label>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            onClick={() => void props.onSubmit({ name, description })}
            disabled={!name.trim() || props.isPending}
          >
            {t('Create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

type EditGroupDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  initialName: string
  initialDescription: string
  isPending: boolean
  onSubmit: (values: { name: string; description: string }) => Promise<void>
}

export function EditGroupDialog(props: EditGroupDialogProps) {
  const { t } = useTranslation()
  const [name, setName] = useState(props.initialName)
  const [description, setDescription] = useState(props.initialDescription)

  useEffect(() => {
    if (props.open) {
      setName(props.initialName)
      setDescription(props.initialDescription)
    }
  }, [props.open, props.initialName, props.initialDescription])

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('Edit asset group')}</DialogTitle>
        </DialogHeader>
        <div className='space-y-3'>
          <div className='space-y-1'>
            <Label>{t('Name')}</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className='space-y-1'>
            <Label>{t('Description')}</Label>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            onClick={() => void props.onSubmit({ name, description })}
            disabled={!name.trim() || props.isPending}
          >
            {t('Save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
