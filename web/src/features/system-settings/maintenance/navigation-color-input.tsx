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
import { RotateCcw } from 'lucide-react'
import type { Ref } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { FormControl } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { normalizeNavColor } from '@/lib/navigation-colors'

type NavigationColorInputProps = {
  label: string
  value: string
  name: string
  onChange: (value: string) => void
  onBlur: () => void
  inputRef: Ref<HTMLInputElement>
}

export function NavigationColorInput(props: NavigationColorInputProps) {
  const { t } = useTranslation()

  return (
    <div className='flex min-w-0 items-center gap-2'>
      <input
        type='color'
        value={normalizeNavColor(props.value) ?? '#6B7280'}
        onChange={(event) => props.onChange(event.target.value)}
        className='border-input h-9 w-10 shrink-0 cursor-pointer rounded-md border bg-transparent p-0.5'
        aria-label={props.label}
      />
      <FormControl>
        <Input
          ref={props.inputRef}
          name={props.name}
          value={props.value}
          onChange={(event) => props.onChange(event.target.value)}
          onBlur={props.onBlur}
          placeholder='#RRGGBB'
          maxLength={7}
          className='min-w-0'
        />
      </FormControl>
      <Button
        type='button'
        size='icon-sm'
        variant='ghost'
        disabled={!props.value}
        onClick={() => props.onChange('')}
        title={t('Reset')}
        aria-label={t('Reset')}
      >
        <RotateCcw />
      </Button>
    </div>
  )
}
