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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2, Pencil, Plus, Search, Trash2 } from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { getLobeIcon } from '@/lib/lobe-icon'

import { deleteVendor, searchVendors } from '../../api'
import { vendorsQueryKeys } from '../../lib'
import type { Vendor } from '../../types'

const PAGE_SIZE = 20

type VendorManagementDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreate: () => void
  onEdit: (vendor: Vendor) => void
}

export function VendorManagementDialog(props: VendorManagementDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [deleteTarget, setDeleteTarget] = useState<Vendor | null>(null)

  const vendorsQuery = useQuery({
    queryKey: vendorsQueryKeys.list({
      keyword: search,
      p: page,
      page_size: PAGE_SIZE,
    }),
    queryFn: () =>
      searchVendors({ keyword: search, p: page, page_size: PAGE_SIZE }),
    enabled: props.open,
  })
  const vendors = vendorsQuery.data?.data?.items ?? []
  const total = vendorsQuery.data?.data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const deleteMutation = useMutation({
    mutationFn: deleteVendor,
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Failed to delete vendor'))
        return
      }
      toast.success(t('Vendor deleted successfully'))
      setDeleteTarget(null)
      if (vendors.length === 1 && page > 1) setPage(page - 1)
      void queryClient.invalidateQueries({ queryKey: vendorsQueryKeys.lists() })
    },
    onError: (error) => {
      toast.error(error.message || t('Failed to delete vendor'))
    },
  })

  let vendorContent: ReactNode
  if (vendorsQuery.isPending) {
    vendorContent = (
      <div className='flex justify-center py-10' role='status'>
        <Loader2 className='h-5 w-5 animate-spin' />
        <span className='sr-only'>{t('Loading...')}</span>
      </div>
    )
  } else if (vendorsQuery.isError || !vendorsQuery.data?.success) {
    vendorContent = (
      <p className='text-destructive py-10 text-center text-sm'>
        {vendorsQuery.error?.message ||
          vendorsQuery.data?.message ||
          t('Failed to load')}
      </p>
    )
  } else if (vendors.length === 0) {
    vendorContent = (
      <p className='text-muted-foreground py-10 text-center text-sm'>
        {t('No data')}
      </p>
    )
  } else {
    vendorContent = (
      <div className='divide-y rounded-md border'>
        {vendors.map((vendor) => (
          <div key={vendor.id} className='flex items-center gap-3 px-3 py-2'>
            <span className='flex h-8 w-8 shrink-0 items-center justify-center'>
              {getLobeIcon(vendor.icon || vendor.name, 24)}
            </span>
            <div className='min-w-0 flex-1'>
              <div className='truncate text-sm font-medium'>{vendor.name}</div>
              {vendor.description ? (
                <div className='text-muted-foreground truncate text-xs'>
                  {vendor.description}
                </div>
              ) : null}
            </div>
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    size='icon-sm'
                    variant='ghost'
                    onClick={() => props.onEdit(vendor)}
                    aria-label={`${t('Edit Vendor')}: ${vendor.name}`}
                  />
                }
              >
                <Pencil className='h-4 w-4' />
              </TooltipTrigger>
              <TooltipContent>{t('Edit Vendor')}</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger
                render={
                  <Button
                    size='icon-sm'
                    variant='ghost'
                    className='text-destructive hover:text-destructive'
                    onClick={() => setDeleteTarget(vendor)}
                    aria-label={`${t('Delete Vendor')}: ${vendor.name}`}
                  />
                }
              >
                <Trash2 className='h-4 w-4' />
              </TooltipTrigger>
              <TooltipContent>{t('Delete Vendor')}</TooltipContent>
            </Tooltip>
          </div>
        ))}
      </div>
    )
  }

  return (
    <>
      <Dialog
        open={props.open}
        onOpenChange={props.onOpenChange}
        title={t('Manage Vendors')}
        contentClassName='sm:max-w-xl'
        contentHeight='min(55vh, 460px)'
        footer={
          <Button size='sm' onClick={props.onCreate}>
            <Plus className='h-4 w-4' />
            {t('Create Vendor')}
          </Button>
        }
      >
        <div className='space-y-3'>
          <div className='relative'>
            <Search className='text-muted-foreground pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2' />
            <Input
              aria-label={t('Search vendors...')}
              placeholder={t('Search vendors...')}
              value={search}
              onChange={(event) => {
                setSearch(event.target.value)
                setPage(1)
              }}
              className='pl-9'
            />
          </div>
          <p className='text-muted-foreground text-xs'>
            {t(
              'Reassign models before deleting their vendor. Icons remain available.'
            )}
          </p>
          {vendorContent}
          <div className='flex items-center justify-between gap-2 text-sm'>
            <span className='text-muted-foreground'>
              {t('{{count}} vendors', { count: total })}
            </span>
            <div className='flex items-center gap-2'>
              <Button
                size='sm'
                variant='outline'
                disabled={page <= 1 || vendorsQuery.isFetching}
                onClick={() => setPage(page - 1)}
              >
                {t('Previous')}
              </Button>
              <span className='text-muted-foreground whitespace-nowrap'>
                {t('Page {{current}} of {{total}}', {
                  current: page,
                  total: pageCount,
                })}
              </span>
              <Button
                size='sm'
                variant='outline'
                disabled={page >= pageCount || vendorsQuery.isFetching}
                onClick={() => setPage(page + 1)}
              >
                {t('Next')}
              </Button>
            </div>
          </div>
        </div>
      </Dialog>
      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open && !deleteMutation.isPending) setDeleteTarget(null)
        }}
        title={t('Delete Vendor')}
        desc={t(
          'Delete vendor "{{name}}"? The icon library will remain available.',
          {
            name: deleteTarget?.name ?? '',
          }
        )}
        confirmText={t('Delete')}
        destructive
        isLoading={deleteMutation.isPending}
        handleConfirm={() => {
          if (deleteTarget) deleteMutation.mutate(deleteTarget.id)
        }}
      />
    </>
  )
}
