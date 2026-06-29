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
import { useContext, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group'
import {
  SettingsControlGroup,
  SettingsSwitchField,
} from '../components/settings-form-layout'
import { PriceCurrencyContext } from './price-currency'

// 存储精度（写入美元价 10 位）与显示精度（人民币回显 8 位，消除换算尾噪）
const fmtStore = (n: number) =>
  Number.isFinite(n) ? String(parseFloat(n.toFixed(10))) : ''
const fmtDisplay = (n: number) =>
  Number.isFinite(n) ? String(parseFloat(n.toFixed(8))) : ''

export function PriceInput(props: {
  value: string
  placeholder?: string
  disabled?: boolean
  onChange: (value: string) => void
}) {
  const { isRMB, rate } = useContext(PriceCurrencyContext)
  // 人民币模式下保留用户原始输入（含未完成的小数点），避免换算把 "31." 吞掉
  const [draft, setDraft] = useState('')
  const selfEdit = useRef(false)

  useEffect(() => {
    if (!isRMB) return
    if (selfEdit.current) {
      selfEdit.current = false
      return
    }
    setDraft(
      props.value !== '' && props.value != null
        ? fmtDisplay(Number(props.value) * rate)
        : ''
    )
  }, [props.value, isRMB, rate])

  const handleChange = (raw: string) => {
    if (!isRMB) {
      props.onChange(raw)
      return
    }
    setDraft(raw)
    selfEdit.current = true
    if (raw === '' || raw == null) {
      props.onChange('')
      return
    }
    const num = Number(raw)
    props.onChange(Number.isFinite(num) ? fmtStore(num / rate) : '')
  }

  const value = isRMB ? draft : props.value
  const symbol = isRMB ? '¥' : '$'
  const suffix = isRMB ? '¥/1M' : '$/1M'

  return (
    <InputGroup>
      <InputGroupAddon>{symbol}</InputGroupAddon>
      <InputGroupInput
        inputMode='decimal'
        value={value}
        placeholder={props.placeholder}
        disabled={props.disabled}
        onChange={(event) => handleChange(event.target.value)}
      />
      <InputGroupAddon align='inline-end'>{suffix}</InputGroupAddon>
    </InputGroup>
  )
}

export function PriceLane(props: {
  title: string
  description: string
  placeholder: string
  value: string
  enabled: boolean
  disabled?: boolean
  onEnabledChange: (checked: boolean) => void
  onChange: (value: string) => void
}) {
  const { t } = useTranslation()
  const effectiveDisabled = props.disabled || !props.enabled

  return (
    <SettingsControlGroup
      className={cn('space-y-3', effectiveDisabled && 'opacity-75')}
      data-disabled={effectiveDisabled || undefined}
    >
      <SettingsSwitchField
        checked={props.enabled}
        disabled={props.disabled}
        onCheckedChange={props.onEnabledChange}
        label={props.title}
        description={props.description}
        aria-label={props.title}
      />
      <PriceInput
        value={props.value}
        placeholder={props.placeholder}
        disabled={effectiveDisabled}
        onChange={props.onChange}
      />
      <p className='text-muted-foreground text-xs'>
        {props.enabled
          ? t('USD price per 1M tokens.')
          : t('Disabled lanes are omitted on save.')}
      </p>
    </SettingsControlGroup>
  )
}
