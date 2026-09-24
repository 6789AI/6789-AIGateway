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
import type { ColumnDef } from '@tanstack/react-table'
import { useTranslation } from 'react-i18next'

import { BadgeCell } from '@/components/data-table'
import { GroupBadge } from '@/components/group-badge'
import { LongText } from '@/components/long-text'
import { StatusBadge } from '@/components/status-badge'
import { TableId } from '@/components/table-id'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatQuota, formatTimestamp } from '@/lib/format'

import { USER_ROLES, getUserStatusConfig } from '../constants'
import type { User } from '../types'
import { DataTableRowActions } from './data-table-row-actions'
import { UserQuotaCell } from './user-quota-cell'
import { UserStatusIndicator } from './user-status-indicator'

export function useUsersColumns(): ColumnDef<User>[] {
  const { t } = useTranslation()
  return [
    {
      id: 'select',
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          indeterminate={table.getIsSomePageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label='Select all'
          className='translate-y-[2px]'
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label='Select row'
          className='translate-y-[2px]'
        />
      ),
      enableSorting: false,
      enableHiding: false,
      size: 40,
    },
    {
      accessorKey: 'id',
      header: t('ID'),
      cell: ({ row, table }) => {
        const isStatusHidden = !table.getColumn('status')?.getIsVisible()
        const isUsernameHidden = !table.getColumn('username')?.getIsVisible()

        return (
          <div className='flex w-fit items-end gap-1'>
            <TableId
              value={row.getValue('id') as number}
              className='w-[60px] text-sm'
            />
            {isStatusHidden && isUsernameHidden ? (
              <UserStatusIndicator user={row.original} />
            ) : null}
          </div>
        )
      },
      size: 80,
      meta: { mobileOrder: 10 },
    },
    {
      accessorKey: 'username',
      header: t('Username'),
      cell: ({ row, table }) => {
        const username = row.getValue('username') as string
        const displayName = row.original.display_name
        const remark = row.original.remark
        const isStatusHidden = !table.getColumn('status')?.getIsVisible()

        return (
          <div className='flex min-w-[160px] flex-col gap-1'>
            <div className='flex items-center gap-2'>
              <div className='flex min-w-0 items-end gap-1'>
                <LongText className='max-w-[140px] font-medium'>
                  {username}
                </LongText>
                {isStatusHidden ? (
                  <UserStatusIndicator user={row.original} />
                ) : null}
              </div>
              {remark && (
                <Tooltip>
                  <TooltipTrigger
                    render={<StatusBadge variant='success' copyable={false} />}
                  >
                    <LongText className='max-w-[80px]'>{remark}</LongText>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p className='text-xs'>{remark}</p>
                  </TooltipContent>
                </Tooltip>
              )}
            </div>
            {displayName && displayName !== username && (
              <LongText className='text-muted-foreground max-w-[180px] text-xs'>
                {displayName}
              </LongText>
            )}
          </div>
        )
      },
      size: 220,
      meta: { mobileTitle: true },
    },
    {
      accessorKey: 'email',
      header: t('Email'),
      cell: ({ row }) => {
        const email = row.getValue('email') as string | undefined

        return email ? (
          <LongText className='max-w-[240px] text-sm'>{email}</LongText>
        ) : (
          <span className='text-muted-foreground text-sm'>-</span>
        )
      },
      enableSorting: false,
      size: 260,
      meta: { mobileOrder: 25 },
    },
    {
      accessorKey: 'status',
      header: t('Status'),
      cell: ({ row }) => {
        const user = row.original
        const requestCount = user.request_count

        const statusConfig = getUserStatusConfig(user)

        if (!statusConfig) {
          return null
        }

        return (
          <Tooltip>
            <TooltipTrigger render={<div className='-ml-1.5 cursor-help' />}>
              <StatusBadge
                label={t(statusConfig.labelKey)}
                variant={statusConfig.variant}
                copyable={false}
              />
            </TooltipTrigger>
            <TooltipContent>
              <p className='text-xs'>
                {t('Requests:')} {requestCount.toLocaleString()}
              </p>
            </TooltipContent>
          </Tooltip>
        )
      },
      filterFn: (row, id, value) => {
        return value.includes(String(row.getValue(id)))
      },
      enableSorting: false,
      size: 120,
      meta: { mobileBadge: true },
    },
    {
      id: 'quota',
      accessorKey: 'quota',
      header: t('Quota'),
      cell: ({ row }) => {
        const user = row.original
        return <UserQuotaCell used={user.used_quota} remaining={user.quota} />
      },
      size: 300,
      minSize: 260,
      meta: { mobileOrder: 40 },
    },
    {
      accessorKey: 'group',
      header: t('Group'),
      cell: ({ row }) => {
        const group = row.getValue('group') as string
        return (
          <BadgeCell>
            <GroupBadge group={group} />
          </BadgeCell>
        )
      },
      filterFn: (row, id, value) => {
        const group = String(row.getValue(id) || t('User Group')).toLowerCase()
        const searchValue = String(value).toLowerCase()
        return group.includes(searchValue)
      },
      size: 140,
      meta: { mobileOrder: 30 },
    },
    {
      accessorKey: 'role',
      header: t('Role'),
      cell: ({ row }) => {
        const roleValue = row.getValue('role') as number
        const roleConfig = USER_ROLES[roleValue as keyof typeof USER_ROLES]

        if (!roleConfig) {
          return null
        }

        return (
          <div className='flex items-center gap-x-2'>
            {roleConfig.icon && (
              <roleConfig.icon size={16} className='text-muted-foreground' />
            )}
            <span className='text-sm'>{t(roleConfig.labelKey)}</span>
          </div>
        )
      },
      filterFn: (row, id, value) => {
        return value.includes(String(row.getValue(id)))
      },
      enableSorting: false,
      size: 120,
      meta: { mobileOrder: 20 },
    },
    {
      id: 'invite_info',
      header: t('Invite Info'),
      cell: ({ row }) => {
        const user = row.original
        const affCount = user.aff_count || 0
        const affHistoryQuota = user.aff_history_quota || 0
        const inviterId = user.inviter_id || 0

        return (
          <div className='flex max-w-full min-w-0 flex-wrap items-center gap-1 overflow-hidden'>
            <Tooltip>
              <TooltipTrigger
                render={
                  <StatusBadge
                    label={`${t('Invited')}: ${affCount}`}
                    variant='neutral'
                    copyable={false}
                    className='cursor-help'
                  />
                }
              />
              <TooltipContent>
                <p className='text-xs'>{t('Number of users invited')}</p>
              </TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger
                render={
                  <StatusBadge
                    label={`${t('Revenue')}: ${formatQuota(affHistoryQuota)}`}
                    variant='neutral'
                    copyable={false}
                    className='cursor-help'
                  />
                }
              />
              <TooltipContent>
                <p className='text-xs'>{t('Total invitation revenue')}</p>
              </TooltipContent>
            </Tooltip>
            {inviterId > 0 && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <StatusBadge
                      label={`${t('Inviter')}: ${inviterId}`}
                      variant='neutral'
                      copyable={false}
                      className='cursor-help'
                    />
                  }
                />
                <TooltipContent>
                  <p className='text-xs'>
                    {t('Invited by user ID')} {inviterId}
                  </p>
                </TooltipContent>
              </Tooltip>
            )}
            {inviterId === 0 && (
              <StatusBadge
                label={t('No Inviter')}
                variant='neutral'
                copyable={false}
              />
            )}
          </div>
        )
      },
      size: 240,
      enableSorting: false,
      meta: { mobileHidden: true },
    },
    {
      accessorKey: 'created_at',
      header: t('Created At'),
      cell: ({ row }) => {
        const ts = row.getValue('created_at') as number | undefined
        return (
          <span className='text-muted-foreground text-sm'>
            {ts ? formatTimestamp(ts) : '-'}
          </span>
        )
      },
      size: 180,
      meta: { mobileHidden: true },
    },
    {
      accessorKey: 'last_login_at',
      header: t('Last Login'),
      cell: ({ row }) => {
        const ts = row.getValue('last_login_at') as number | undefined
        return (
          <span className='text-muted-foreground text-sm'>
            {ts ? formatTimestamp(ts) : '-'}
          </span>
        )
      },
      size: 180,
      meta: { mobileHidden: true },
    },
    {
      id: 'actions',
      header: () => t('Actions'),
      cell: ({ row }) => <DataTableRowActions row={row} />,
      meta: { pinned: 'right' as const },
    },
  ]
}
