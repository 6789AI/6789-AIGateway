/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useTranslation } from 'react-i18next'

import { Checkbox } from '@/components/ui/checkbox'

type AuthorizedGroupPickerProps = {
  groups: string[]
  primaryGroup?: string
  value: string[]
  onChange: (groups: string[]) => void
}

export function AuthorizedGroupPicker(props: AuthorizedGroupPickerProps) {
  const { t } = useTranslation()
  const groups = props.groups.filter((group) => group !== props.primaryGroup)
  const deletedGroups = props.value.filter(
    (group) => group !== props.primaryGroup && !props.groups.includes(group)
  )
  const atLimit = props.value.length >= 32

  return (
    <div
      role='group'
      aria-label={t('Additional authorized groups')}
      className='max-h-44 space-y-1 overflow-y-auto rounded-md border p-2'
    >
      {[...groups, ...deletedGroups].map((group) => {
        const deleted = deletedGroups.includes(group)
        const checked = props.value.includes(group)
        return (
          <label
            key={group}
            className='hover:bg-muted/50 flex min-h-9 items-center gap-2 rounded px-2'
          >
            <Checkbox
              checked={checked}
              disabled={!checked && atLimit}
              onCheckedChange={(checked) => {
                props.onChange(
                  checked === true
                    ? [...props.value, group]
                    : props.value.filter((item) => item !== group)
                )
              }}
            />
            <span className='min-w-0 text-sm break-all'>{group}</span>
            {deleted && (
              <span className='text-muted-foreground text-xs'>
                {t('Deleted')}
              </span>
            )}
          </label>
        )
      })}
      {atLimit && (
        <p className='text-muted-foreground px-2 text-xs'>
          {t('Maximum {{max}} groups selected', { max: 32 })}
        </p>
      )}
    </div>
  )
}
