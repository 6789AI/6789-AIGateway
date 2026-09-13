/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
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

import { StatusBadge } from '@/components/status-badge'
import { Badge } from '@/components/ui/badge'
import { formatTimestamp } from '@/lib/format'

import { USER_STATUS, USER_STATUSES } from '../constants'
import type {
  AffiliateLookupInviter,
  AffiliateLookupOwner,
  AffiliateLookupUser,
} from '../types'

type InvitationLookupUserDetailsProps = {
  affCode?: string
  heading: string
  headingId: string
  user: AffiliateLookupUser
}

export function AffiliateLookupStatus(props: { user: AffiliateLookupUser }) {
  const { t } = useTranslation()
  const status = props.user.deleted_at ? USER_STATUS.DELETED : props.user.status
  const config = USER_STATUSES[status as keyof typeof USER_STATUSES]

  if (!config) {
    return (
      <StatusBadge label={t('Unknown')} variant='neutral' copyable={false} />
    )
  }
  return (
    <StatusBadge
      label={t(config.labelKey)}
      variant={config.variant}
      copyable={false}
    />
  )
}

export function InvitationLookupUserDetails(
  props: InvitationLookupUserDetailsProps
) {
  const { t } = useTranslation()

  return (
    <section aria-labelledby={props.headingId} className='flex flex-col gap-3'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <h3 id={props.headingId} className='text-sm font-semibold'>
          {props.heading}
        </h3>
        {props.affCode ? (
          <Badge variant='outline'>{props.affCode}</Badge>
        ) : null}
      </div>
      <dl className='grid gap-x-6 gap-y-3 sm:grid-cols-2 lg:grid-cols-4'>
        <div className='min-w-0'>
          <dt className='text-muted-foreground text-xs'>{t('ID')}</dt>
          <dd className='mt-1 text-sm font-medium tabular-nums'>
            {props.user.id}
          </dd>
        </div>
        <div className='min-w-0'>
          <dt className='text-muted-foreground text-xs'>{t('Username')}</dt>
          <dd className='mt-1 min-w-0 text-sm font-medium break-words'>
            {props.user.username}
            {props.user.display_name &&
            props.user.display_name !== props.user.username ? (
              <span className='text-muted-foreground ms-1 font-normal'>
                ({props.user.display_name})
              </span>
            ) : null}
          </dd>
        </div>
        <div className='min-w-0'>
          <dt className='text-muted-foreground text-xs'>{t('Email')}</dt>
          <dd className='mt-1 text-sm break-all'>{props.user.email || '-'}</dd>
        </div>
        <div className='min-w-0'>
          <dt className='text-muted-foreground text-xs'>{t('Status')}</dt>
          <dd className='mt-1'>
            <AffiliateLookupStatus user={props.user} />
          </dd>
        </div>
        <div className='min-w-0 sm:col-span-2 lg:col-span-4'>
          <dt className='text-muted-foreground text-xs'>{t('Created At')}</dt>
          <dd className='mt-1 text-sm tabular-nums'>
            {props.user.created_at
              ? formatTimestamp(props.user.created_at)
              : '-'}
          </dd>
        </div>
      </dl>
    </section>
  )
}

export function InvitationLookupInviterDetails(props: {
  inviter: AffiliateLookupInviter | null
  owner: AffiliateLookupOwner
}) {
  const { t } = useTranslation()

  if (props.inviter) {
    return (
      <InvitationLookupUserDetails
        user={props.inviter}
        affCode={props.inviter.aff_code}
        heading={t('Invited by')}
        headingId='invitation-inviter-heading'
      />
    )
  }

  return (
    <section
      aria-labelledby='invitation-inviter-heading'
      className='flex flex-col gap-2'
    >
      <h3 id='invitation-inviter-heading' className='text-sm font-semibold'>
        {t('Invited by')}
      </h3>
      <p className='text-muted-foreground text-sm'>
        {props.owner.inviter_id > 0
          ? t('The recorded inviter no longer exists (ID: {{id}}).', {
              id: props.owner.inviter_id,
            })
          : t('This user was not registered through an invitation.')}
      </p>
    </section>
  )
}
