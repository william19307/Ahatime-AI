/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { api } from '@/lib/api'
import type {
  ApiResponse,
  PaginatedResponse,
  SeedanceAsset,
  SeedanceAssetGroup,
  SeedanceUpload,
} from './types'

export async function listSeedanceGroups(page = 1, size = 50) {
  const res = await api.get<PaginatedResponse<SeedanceAssetGroup>>(
    `/api/seedance/groups?p=${page}&size=${size}`
  )
  return res.data
}

export async function createSeedanceGroup(data: {
  name: string
  description?: string
  group_type?: string
}) {
  const res = await api.post<ApiResponse<SeedanceAssetGroup>>(
    '/api/seedance/groups',
    data
  )
  return res.data
}

export async function updateSeedanceGroup(
  id: number,
  data: { name: string; description?: string }
) {
  const res = await api.put<ApiResponse<SeedanceAssetGroup>>(
    `/api/seedance/groups/${id}`,
    data
  )
  return res.data
}

export async function listSeedanceAssets(params: {
  page?: number
  size?: number
  group_id?: number
  keyword?: string
}) {
  const query = new URLSearchParams()
  query.set('p', String(params.page ?? 1))
  query.set('size', String(params.size ?? 20))
  if (params.group_id) query.set('group_id', String(params.group_id))
  if (params.keyword) query.set('keyword', params.keyword)
  const res = await api.get<PaginatedResponse<SeedanceAsset>>(
    `/api/seedance/assets?${query.toString()}`
  )
  return res.data
}

export async function createSeedanceAsset(data: {
  group_id: number
  url?: string
  upload_id?: number
  asset_type: string
  name?: string
}) {
  const res = await api.post<ApiResponse<SeedanceAsset>>(
    '/api/seedance/assets',
    data
  )
  return res.data
}

export async function updateSeedanceAsset(id: number, name: string) {
  const res = await api.put<ApiResponse<SeedanceAsset>>(
    `/api/seedance/assets/${id}`,
    { name }
  )
  return res.data
}

export async function deleteSeedanceAsset(id: number) {
  const res = await api.delete<ApiResponse<null>>(`/api/seedance/assets/${id}`)
  return res.data
}

export async function uploadSeedanceFile(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  const res = await api.post<ApiResponse<SeedanceUpload>>(
    '/api/seedance/uploads',
    formData,
    {
      headers: { 'Content-Type': 'multipart/form-data' },
    }
  )
  return res.data
}
