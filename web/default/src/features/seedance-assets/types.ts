/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

export type SeedanceAssetGroup = {
  id: number
  user_id: number
  upstream_id: string
  name: string
  description: string
  group_type: string
  is_default: boolean
  created_at: number
  updated_at: number
}

export type SeedanceAsset = {
  id: number
  user_id: number
  group_id: number
  upstream_id: string
  name: string
  asset_type: string
  source_url: string
  public_url: string
  status: string
  created_at: number
  updated_at: number
}

export type SeedanceUpload = {
  id: number
  file_name: string
  mime_type: string
  size: number
  public_url: string
  expires_at: number
  signed_token: string
}

export type PaginatedResponse<T> = {
  success: boolean
  message: string
  data: {
    page: number
    page_size: number
    total: number
    items: T[]
  }
}

export type ApiResponse<T> = {
  success: boolean
  message: string
  data: T
}
