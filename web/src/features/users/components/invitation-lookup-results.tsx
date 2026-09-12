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
import {
  ArrowLeft01Icon,
  ArrowRight01Icon,
  UserGroupIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import type { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'

import { StatusBadge } from '@/components/status-badge'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Separator } from '@/components/ui/separator'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatTimestamp } from '@/lib/format'

import { USER_STATUS, USER_STATUSES } from '../constants'
import type { AffiliateLookupResult, AffiliateLookupUser } from '../types'

type InvitationLookupResultsProps = {
  result: AffiliateLookupResult
  owner: AffiliateLookupUser
  isFetching: boolean
  onPageChange: (page: number) => void
}

function AffiliateStatus(props: { user: AffiliateLookupUser; t: TFunction }) {
  const status = props.user.deleted_at ? USER_STATUS.DELETED : props.user.status
  const config = USER_STATUSES[status as keyof typeof USER_STATUSES]

  if (!config) {
    return (
      <StatusBadge
        label={props.t('Unknown')}
        variant='neutral'
        copyable={false}
      />
    )
  }
  return (
    <StatusBadge
      label={props.t(config.labelKey)}
      variant={config.variant}
      copyable={false}
    />
  )
}

function OwnerDetails(props: {
  owner: AffiliateLookupUser
  affCode: string
  t: TFunction
}) {
  return (
    <section aria-labelledby='invitation-owner-heading' className='space-y-3'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <h3 id='invitation-owner-heading' className='text-sm font-semibold'>
          {props.t('Invitation link owner')}
        </h3>
        <Badge variant='outline'>{props.affCode}</Badge>
      </div>
      <dl className='grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-4'>
        <div className='min-w-0'>
          <dt className='text-muted-foreground text-xs'>{props.t('ID')}</dt>
          <dd className='mt-1 text-sm font-medium tabular-nums'>
            {props.owner.id}
          </dd>
        </div>
        <div className='min-w-0'>
          <dt className='text-muted-foreground text-xs'>
            {props.t('Username')}
          </dt>
          <dd className='mt-1 min-w-0 text-sm font-medium break-words'>
            {props.owner.username}
            {props.owner.display_name &&
              props.owner.display_name !== props.owner.username && (
                <span className='text-muted-foreground ms-1 font-normal'>
                  ({props.owner.display_name})
                </span>
              )}
          </dd>
        </div>
        <div className='min-w-0'>
          <dt className='text-muted-foreground text-xs'>{props.t('Email')}</dt>
          <dd className='mt-1 text-sm break-all'>{props.owner.email || '-'}</dd>
        </div>
        <div className='min-w-0'>
          <dt className='text-muted-foreground text-xs'>{props.t('Status')}</dt>
          <dd className='mt-1'>
            <AffiliateStatus user={props.owner} t={props.t} />
          </dd>
        </div>
        <div className='min-w-0 sm:col-span-2 lg:col-span-4'>
          <dt className='text-muted-foreground text-xs'>
            {props.t('Created At')}
          </dt>
          <dd className='mt-1 text-sm tabular-nums'>
            {props.owner.created_at
              ? formatTimestamp(props.owner.created_at)
              : '-'}
          </dd>
        </div>
      </dl>
    </section>
  )
}

export function InvitationLookupResults(props: InvitationLookupResultsProps) {
  const { t } = useTranslation()
  const totalPages = Math.max(
    1,
    Math.ceil(props.result.invitees.total / props.result.invitees.page_size)
  )

  return (
    <div className='space-y-4'>
      <OwnerDetails owner={props.owner} affCode={props.result.aff_code} t={t} />
      <Separator />
      <section aria-labelledby='direct-invitees-heading' className='space-y-3'>
        <div className='flex items-center justify-between gap-2'>
          <h3 id='direct-invitees-heading' className='text-sm font-semibold'>
            {t('Direct invitees')}
          </h3>
          <Badge variant='secondary'>{props.result.invitees.total}</Badge>
        </div>

        {props.result.invitees.items.length === 0 ? (
          <Empty className='min-h-36 border'>
            <EmptyHeader>
              <EmptyMedia variant='icon'>
                <HugeiconsIcon icon={UserGroupIcon} strokeWidth={2} />
              </EmptyMedia>
              <EmptyTitle>{t('No invited users')}</EmptyTitle>
              <EmptyDescription>
                {t('No users were registered with this invitation code.')}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <div className='rounded-lg border'>
            <Table className='min-w-[720px]'>
              <TableHeader>
                <TableRow>
                  <TableHead className='w-20'>{t('ID')}</TableHead>
                  <TableHead>{t('Username')}</TableHead>
                  <TableHead>{t('Email')}</TableHead>
                  <TableHead className='w-28'>{t('Status')}</TableHead>
                  <TableHead className='w-44'>{t('Created At')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {props.result.invitees.items.map((user) => (
                  <TableRow key={user.id}>
                    <TableCell className='tabular-nums'>{user.id}</TableCell>
                    <TableCell>
                      <div className='max-w-48 whitespace-normal'>
                        <div className='font-medium break-words'>
                          {user.username}
                        </div>
                        {user.display_name &&
                          user.display_name !== user.username && (
                            <div className='text-muted-foreground text-xs break-words'>
                              {user.display_name}
                            </div>
                          )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <span className='block max-w-56 break-all whitespace-normal'>
                        {user.email || '-'}
                      </span>
                    </TableCell>
                    <TableCell>
                      <AffiliateStatus user={user} t={t} />
                    </TableCell>
                    <TableCell className='tabular-nums'>
                      {user.created_at ? formatTimestamp(user.created_at) : '-'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}

        {props.result.invitees.total > props.result.invitees.page_size && (
          <div className='flex items-center justify-between gap-3'>
            <p className='text-muted-foreground text-xs tabular-nums'>
              {t('Page {{page}} of {{total}}', {
                page: props.result.invitees.page,
                total: totalPages,
              })}
            </p>
            <div className='flex items-center gap-1'>
              <Button
                type='button'
                variant='outline'
                size='icon-sm'
                title={t('Previous')}
                aria-label={t('Previous')}
                disabled={props.result.invitees.page <= 1 || props.isFetching}
                onClick={() =>
                  props.onPageChange(props.result.invitees.page - 1)
                }
              >
                <HugeiconsIcon icon={ArrowLeft01Icon} strokeWidth={2} />
              </Button>
              <Button
                type='button'
                variant='outline'
                size='icon-sm'
                title={t('Next')}
                aria-label={t('Next')}
                disabled={
                  props.result.invitees.page >= totalPages || props.isFetching
                }
                onClick={() =>
                  props.onPageChange(props.result.invitees.page + 1)
                }
              >
                <HugeiconsIcon icon={ArrowRight01Icon} strokeWidth={2} />
              </Button>
            </div>
          </div>
        )}
      </section>
    </div>
  )
}
