/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useCallback, useState } from 'react'
import { getRouteApi } from '@tanstack/react-router'
import { Download } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'

const route = getRouteApi('/_authenticated/usage-logs/$section')

function msToSeconds(value: unknown) {
  if (value === undefined || value === null || value === '') return undefined
  const timestamp = Number(value)
  if (!Number.isFinite(timestamp)) return undefined
  return Math.floor(timestamp / 1000)
}

function getDownloadFilename(disposition: string | undefined) {
  const filenameMatch =
    disposition?.match(/filename\*=UTF-8''([^;]+)/) ||
    disposition?.match(/filename="?([^";]+)"?/)

  return filenameMatch?.[1]
    ? decodeURIComponent(filenameMatch[1])
    : 'monthly-usage-report.xls'
}

export function CommonLogsExportAction() {
  const { t } = useTranslation()
  const searchParams = route.useSearch()
  const [exporting, setExporting] = useState(false)

  const handleExportMonthlyReport = useCallback(async () => {
    const query = new URLSearchParams()
    const startTimestamp = msToSeconds(searchParams.startTime)
    const endTimestamp = msToSeconds(searchParams.endTime)

    if (startTimestamp !== undefined) {
      query.set('start_timestamp', String(startTimestamp))
    }
    if (endTimestamp !== undefined) {
      query.set('end_timestamp', String(endTimestamp))
    }
    if (searchParams.model) query.set('model_name', searchParams.model)
    if (searchParams.token) query.set('token_name', searchParams.token)
    if (searchParams.group) query.set('group', searchParams.group)

    const suffix = query.toString()
    const url = `/api/log/self/monthly_report${suffix ? `?${suffix}` : ''}`

    setExporting(true)
    try {
      const response = await api.get(url, {
        responseType: 'blob',
        disableDuplicate: true,
        skipErrorHandler: true,
      })
      const blob = response.data
      const filename = getDownloadFilename(
        response.headers['content-disposition'] as string | undefined
      )
      const downloadUrl = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = downloadUrl
      link.download = filename
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(downloadUrl)
    } catch {
      toast.error(t('导出失败，请刷新页面后重试'))
    } finally {
      setExporting(false)
    }
  }, [
    searchParams.endTime,
    searchParams.group,
    searchParams.model,
    searchParams.startTime,
    searchParams.token,
    t,
  ])

  return (
    <Button
      type='button'
      variant='outline'
      size='sm'
      onClick={handleExportMonthlyReport}
      disabled={exporting}
      className='gap-1.5'
    >
      <Download className='size-3.5' />
      {exporting ? t('导出中...') : t('导出当前筛选')}
    </Button>
  )
}
