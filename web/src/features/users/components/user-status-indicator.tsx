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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { dotColorMap, type StatusVariant } from '@/components/status-badge'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import { getUserStatusConfig } from '../constants'
import type { User } from '../types'

function UserStatusDot(props: {
  label: string
  status: number
  variant: StatusVariant
}) {
  const [open, setOpen] = useState(false)

  return (
    <Tooltip open={open} onOpenChange={setOpen}>
      <TooltipTrigger
        render={
          <button
            type='button'
            aria-label={props.label}
            data-user-status={props.status}
            className='focus-visible:ring-ring flex size-4 shrink-0 cursor-help items-end justify-end rounded-full focus-visible:ring-2 focus-visible:outline-none'
            onClick={() => setOpen(true)}
          />
        }
      >
        <span
          className={cn('size-2 rounded-full', dotColorMap[props.variant])}
          aria-hidden='true'
        />
      </TooltipTrigger>
      <TooltipContent>
        <p>{props.label}</p>
      </TooltipContent>
    </Tooltip>
  )
}

export function UserStatusIndicator(props: { user: User }) {
  const { t } = useTranslation()
  const statusConfig = getUserStatusConfig(props.user)

  if (!statusConfig) {
    return null
  }

  return (
    <UserStatusDot
      label={t(statusConfig.labelKey)}
      status={statusConfig.value}
      variant={statusConfig.variant}
    />
  )
}
