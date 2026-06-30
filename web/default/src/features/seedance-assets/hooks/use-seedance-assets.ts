/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  createSeedanceAsset,
  createSeedanceGroup,
  deleteSeedanceAsset,
  listSeedanceAssets,
  listSeedanceGroups,
  updateSeedanceAsset,
  updateSeedanceGroup,
  uploadSeedanceFile,
} from '../api'

export function useSeedanceGroups() {
  return useQuery({
    queryKey: ['seedance-groups'],
    queryFn: async () => {
      const res = await listSeedanceGroups()
      if (!res.success) throw new Error(res.message)
      return res.data.items
    },
  })
}

export function useSeedanceAssets(groupId?: number, keyword?: string) {
  return useQuery({
    queryKey: ['seedance-assets', groupId, keyword],
    queryFn: async () => {
      const res = await listSeedanceAssets({
        group_id: groupId,
        keyword,
        size: 50,
      })
      if (!res.success) throw new Error(res.message)
      return res.data
    },
  })
}

export function useSeedanceAssetMutations() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const invalidate = async () => {
    await queryClient.invalidateQueries({ queryKey: ['seedance-groups'] })
    await queryClient.invalidateQueries({ queryKey: ['seedance-assets'] })
  }

  const createGroup = useMutation({
    mutationFn: createSeedanceGroup,
    onSuccess: async (res) => {
      if (!res.success) {
        toast.error(res.message)
        return
      }
      toast.success(t('Asset group created'))
      await invalidate()
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const updateGroup = useMutation({
    mutationFn: ({
      id,
      name,
      description,
    }: {
      id: number
      name: string
      description?: string
    }) => updateSeedanceGroup(id, { name, description }),
    onSuccess: async (res) => {
      if (!res.success) {
        toast.error(res.message)
        return
      }
      toast.success(t('Asset group updated'))
      await invalidate()
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const createAsset = useMutation({
    mutationFn: createSeedanceAsset,
    onSuccess: async (res) => {
      if (!res.success) {
        toast.error(res.message)
        return
      }
      toast.success(t('Asset created'))
      await invalidate()
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const updateAsset = useMutation({
    mutationFn: ({ id, name }: { id: number; name: string }) =>
      updateSeedanceAsset(id, name),
    onSuccess: async (res) => {
      if (!res.success) {
        toast.error(res.message)
        return
      }
      toast.success(t('Asset updated'))
      await invalidate()
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const removeAsset = useMutation({
    mutationFn: deleteSeedanceAsset,
    onSuccess: async (res) => {
      if (!res.success) {
        toast.error(res.message)
        return
      }
      toast.success(t('Asset deleted'))
      await invalidate()
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const uploadFile = useMutation({
    mutationFn: uploadSeedanceFile,
    onError: (err: Error) => toast.error(err.message),
  })

  return {
    createGroup,
    updateGroup,
    createAsset,
    updateAsset,
    removeAsset,
    uploadFile,
  }
}
