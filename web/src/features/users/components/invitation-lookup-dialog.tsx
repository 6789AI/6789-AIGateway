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
import { Search01Icon, UserSearch02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useQuery } from '@tanstack/react-query'
import { type ReactNode, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'

import { lookupAffiliateUsers } from '../api'
import { InvitationLookupResults } from './invitation-lookup-results'

const INVITEE_PAGE_SIZE = 20

type InvitationLookupDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

type LookupRequest = {
  query: string
  page: number
  requestId: number
}

export function InvitationLookupDialog(props: InvitationLookupDialogProps) {
  const { t } = useTranslation()
  const requestId = useRef(0)
  const [draft, setDraft] = useState('')
  const [request, setRequest] = useState<LookupRequest | null>(null)

  const lookupQuery = useQuery({
    queryKey: [
      'users',
      'affiliate-lookup',
      request?.query,
      request?.page,
      request?.requestId,
    ],
    enabled: props.open && request !== null,
    queryFn: async () => {
      if (!request) {
        throw new Error(t('Failed to search invitation relationships'))
      }
      const response = await lookupAffiliateUsers({
        q: request.query,
        p: request.page,
        page_size: INVITEE_PAGE_SIZE,
      })
      if (!response.success || !response.data) {
        throw new Error(
          response.message || t('Failed to search invitation relationships')
        )
      }
      return response.data
    },
    retry: false,
    staleTime: 0,
    gcTime: 0,
    refetchOnWindowFocus: false,
  })

  const submittedQueryStillMatches = draft.trim() === request?.query
  const result = submittedQueryStillMatches ? lookupQuery.data : undefined
  const error = submittedQueryStillMatches ? lookupQuery.error : null

  const runLookup = (page: number) => {
    const query = draft.trim()
    if (!query) return
    requestId.current += 1
    setRequest({ query, page, requestId: requestId.current })
  }

  const handleClose = (open: boolean) => {
    if (!open) {
      setDraft('')
      setRequest(null)
    }
    props.onOpenChange(open)
  }

  let lookupContent: ReactNode
  if (!request || !submittedQueryStillMatches) {
    lookupContent = (
      <Empty className='min-h-64 border'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <HugeiconsIcon icon={UserSearch02Icon} strokeWidth={2} />
          </EmptyMedia>
          <EmptyTitle>{t('Invitation lookup')}</EmptyTitle>
          <EmptyDescription>
            {t(
              'Search an invitation code or promotion link to view its owner and direct invitees.'
            )}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  } else if (lookupQuery.isLoading) {
    lookupContent = (
      <div className='text-muted-foreground flex min-h-64 items-center justify-center gap-2 text-sm'>
        <Spinner />
        {t('Loading...')}
      </div>
    )
  } else if (result?.owner) {
    lookupContent = (
      <InvitationLookupResults
        result={result}
        owner={result.owner}
        isFetching={lookupQuery.isFetching}
        onPageChange={runLookup}
      />
    )
  } else if (result) {
    lookupContent = (
      <Empty className='min-h-64 border'>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <HugeiconsIcon icon={UserSearch02Icon} strokeWidth={2} />
          </EmptyMedia>
          <EmptyTitle>{t('No invitation relationship found')}</EmptyTitle>
          <EmptyDescription>
            {t('Check the invitation code or promotion link and try again.')}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    )
  } else {
    lookupContent = null
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={handleClose}
      title={t('Invitation lookup')}
      description={t('Search by invitation code or promotion link.')}
      contentClassName='sm:max-w-4xl'
      contentHeight='min(68vh, 640px)'
      bodyClassName='space-y-4'
    >
      <form
        className='flex flex-col gap-2 sm:flex-row sm:items-end'
        onSubmit={(event) => {
          event.preventDefault()
          runLookup(1)
        }}
      >
        <div className='min-w-0 flex-1 space-y-2'>
          <Label htmlFor='affiliate-lookup-query'>
            {t('Invitation code or promotion link')}
          </Label>
          <Input
            id='affiliate-lookup-query'
            value={draft}
            autoComplete='off'
            placeholder={t('Enter an invitation code or promotion link')}
            onChange={(event) => setDraft(event.target.value)}
          />
        </div>
        <Button
          type='submit'
          className='sm:w-28'
          disabled={!draft.trim() || lookupQuery.isFetching}
        >
          {lookupQuery.isFetching ? (
            <Spinner data-icon='inline-start' />
          ) : (
            <HugeiconsIcon
              icon={Search01Icon}
              strokeWidth={2}
              data-icon='inline-start'
            />
          )}
          {t('Search')}
        </Button>
      </form>

      {error && (
        <Alert variant='destructive'>
          <AlertTitle>
            {t('Failed to search invitation relationships')}
          </AlertTitle>
          <AlertDescription>
            {error instanceof Error ? error.message : String(error)}
          </AlertDescription>
        </Alert>
      )}

      {lookupContent}
    </Dialog>
  )
}
