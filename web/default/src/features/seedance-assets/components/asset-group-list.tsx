/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { Spinner } from '@/components/ui/spinner'
import type { SeedanceAssetGroup } from '../types'

type AssetGroupListProps = {
  groups: SeedanceAssetGroup[]
  activeGroupId?: number
  isLoading: boolean
  onSelect: (groupId: number) => void
  onEdit: (group: SeedanceAssetGroup) => void
}

export function AssetGroupList(props: AssetGroupListProps) {
  const { t } = useTranslation()

  return (
    <aside className='border-border w-64 shrink-0 rounded-lg border p-3'>
      <p className='text-muted-foreground mb-2 text-sm font-medium'>
        {t('Asset groups')}
      </p>
      {props.isLoading ? (
        <Spinner className='mx-auto' />
      ) : (
        <div className='space-y-1'>
          {props.groups.map((group) => (
            <div
              key={group.id}
              className={`hover:bg-muted flex items-center gap-1 rounded-md px-1 ${
                props.activeGroupId === group.id ? 'bg-muted' : ''
              }`}
            >
              <button
                type='button'
                className='min-w-0 flex-1 rounded-md px-2 py-2 text-left text-sm'
                onClick={() => props.onSelect(group.id)}
              >
                <div className='flex items-center justify-between gap-2'>
                  <span className='truncate font-medium'>{group.name}</span>
                  {group.is_default ? (
                    <Badge variant='secondary'>{t('Default')}</Badge>
                  ) : null}
                </div>
              </button>
              {!group.is_default ? (
                <button
                  type='button'
                  className='text-muted-foreground hover:text-foreground shrink-0 px-2 py-2 text-xs'
                  onClick={() => props.onEdit(group)}
                >
                  {t('Edit')}
                </button>
              ) : null}
            </div>
          ))}
        </div>
      )}
    </aside>
  )
}
