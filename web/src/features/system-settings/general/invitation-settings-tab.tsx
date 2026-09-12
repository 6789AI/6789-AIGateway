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
import { ReloadIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { type ChangeEvent, useState } from 'react'
import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { formatQuota } from '@/lib/format'

import { recalculateAffiliateInviteCounts } from '../api'
import {
  SettingsFormGrid,
  SettingsFormGridItem,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import type { InviteCountRecalculationResult } from '../types'
import type { QuotaFormValues, QuotaInputValue } from './quota-settings-schema'

type InvitationSettingsTabProps = {
  control: Control<QuotaFormValues>
  complianceConfirmed: boolean
  settingsPending: boolean
}

function quotaDescription(value: QuotaInputValue): string {
  return formatQuota(value === '' ? 0 : value)
}

export function InvitationSettingsTab(props: InvitationSettingsTabProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [lastResult, setLastResult] =
    useState<InviteCountRecalculationResult | null>(null)
  const handleNumberChange =
    (onChange: (value: QuotaInputValue) => void) =>
    (event: ChangeEvent<HTMLInputElement>) => {
      const value = event.currentTarget.valueAsNumber
      onChange(Number.isNaN(value) ? '' : value)
    }

  const recalculateMutation = useMutation({
    mutationFn: async () => {
      const response = await recalculateAffiliateInviteCounts()
      if (!response.success || !response.data) {
        throw new Error(
          response.message || t('Failed to recalculate invitation counts')
        )
      }
      return response.data
    },
    onSuccess: (result) => {
      setLastResult(result)
      setConfirmOpen(false)
      void queryClient.invalidateQueries({ queryKey: ['users'] })
      toast.success(
        t(
          'Recalculated {{relations}} invitation relationships across {{users}} users; updated {{updated}} users.',
          {
            relations: result.invitation_relations,
            users: result.users_scanned,
            updated: result.users_updated,
          }
        )
      )
    },
    onError: (error: Error) => {
      toast.error(error.message || t('Failed to recalculate invitation counts'))
    },
  })

  return (
    <div className='flex min-w-0 flex-col gap-6'>
      {!props.complianceConfirmed ? (
        <Alert variant='destructive'>
          <AlertDescription>
            {t(
              'Non-zero invitation rewards require compliance confirmation in Payment Gateway settings.'
            )}
          </AlertDescription>
        </Alert>
      ) : null}

      <SettingsFormGrid>
        <FormField
          control={props.control}
          name='QuotaForInviterEnabled'
          render={({ field }) => (
            <SettingsSwitchItem>
              <SettingsSwitchContent>
                <FormLabel>{t('Inviter registration reward')}</FormLabel>
                <FormDescription>
                  {t(
                    'Grant a fixed reward to the inviter after a successful referral registration.'
                  )}
                </FormDescription>
              </SettingsSwitchContent>
              <FormControl>
                <Switch
                  checked={field.value}
                  onCheckedChange={field.onChange}
                  disabled={props.settingsPending}
                />
              </FormControl>
            </SettingsSwitchItem>
          )}
        />

        <FormField
          control={props.control}
          name='QuotaForInviter'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Inviter Reward')}</FormLabel>
              <FormControl>
                <Input
                  type='number'
                  min={0}
                  value={field.value ?? ''}
                  onChange={handleNumberChange(field.onChange)}
                  name={field.name}
                  onBlur={field.onBlur}
                  ref={field.ref}
                />
              </FormControl>
              <FormDescription>
                {t('Fixed quota granted to the inviter ({{formattedQuota}})', {
                  formattedQuota: quotaDescription(field.value),
                })}
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={props.control}
          name='QuotaForInviteeEnabled'
          render={({ field }) => (
            <SettingsSwitchItem>
              <SettingsSwitchContent>
                <FormLabel>{t('Invitee registration reward')}</FormLabel>
                <FormDescription>
                  {t(
                    'Grant a fixed reward to a user who registers through a referral.'
                  )}
                </FormDescription>
              </SettingsSwitchContent>
              <FormControl>
                <Switch
                  checked={field.value}
                  onCheckedChange={field.onChange}
                  disabled={props.settingsPending}
                />
              </FormControl>
            </SettingsSwitchItem>
          )}
        />

        <FormField
          control={props.control}
          name='QuotaForInvitee'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Invitee Reward')}</FormLabel>
              <FormControl>
                <Input
                  type='number'
                  min={0}
                  value={field.value ?? ''}
                  onChange={handleNumberChange(field.onChange)}
                  name={field.name}
                  onBlur={field.onBlur}
                  ref={field.ref}
                />
              </FormControl>
              <FormDescription>
                {t('Fixed quota granted to the invitee ({{formattedQuota}})', {
                  formattedQuota: quotaDescription(field.value),
                })}
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={props.control}
          name='AffiliateRebateEnabled'
          render={({ field }) => (
            <SettingsSwitchItem>
              <SettingsSwitchContent>
                <FormLabel>{t('Top-up and redemption rebate')}</FormLabel>
                <FormDescription>
                  {t(
                    'Grant the direct inviter a percentage of successful top-ups and redemption codes.'
                  )}
                </FormDescription>
              </SettingsSwitchContent>
              <FormControl>
                <Switch
                  checked={field.value}
                  onCheckedChange={field.onChange}
                  disabled={props.settingsPending}
                />
              </FormControl>
            </SettingsSwitchItem>
          )}
        />

        <FormField
          control={props.control}
          name='AffiliateRebatePercentage'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Rebate percentage')}</FormLabel>
              <FormControl>
                <Input
                  type='number'
                  min={0}
                  max={100}
                  step={0.01}
                  value={field.value ?? ''}
                  onChange={handleNumberChange(field.onChange)}
                  name={field.name}
                  onBlur={field.onBlur}
                  ref={field.ref}
                />
              </FormControl>
              <FormDescription>
                {t(
                  'Percentage of the final credited quota granted to the direct inviter.'
                )}
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <SettingsFormGridItem span='full'>
          <Separator />
        </SettingsFormGridItem>

        <SettingsFormGridItem span='full'>
          <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
            <div className='min-w-0'>
              <h4 className='text-sm font-medium'>
                {t('Invitation count recalculation')}
              </h4>
              <p className='text-muted-foreground mt-1 text-xs'>
                {t(
                  'Rebuild invitation counts from direct relationships, including soft-deleted users, without changing reward balances.'
                )}
              </p>
              {lastResult ? (
                <p className='text-muted-foreground mt-1 text-xs tabular-nums'>
                  {t(
                    'Last result: {{relations}} relationships, {{users}} users scanned, {{updated}} users updated.',
                    {
                      relations: lastResult.invitation_relations,
                      users: lastResult.users_scanned,
                      updated: lastResult.users_updated,
                    }
                  )}
                </p>
              ) : null}
            </div>

            <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
              <AlertDialogTrigger
                render={<Button type='button' variant='outline' />}
              >
                <HugeiconsIcon
                  icon={ReloadIcon}
                  data-icon='inline-start'
                  strokeWidth={2}
                />
                {t('Recalculate counts')}
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>
                    {t('Recalculate all invitation counts?')}
                  </AlertDialogTitle>
                  <AlertDialogDescription>
                    {t(
                      'This replaces every stored invitation count with the current direct relationships. Pending transfer and lifetime reward totals will not change.'
                    )}
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel disabled={recalculateMutation.isPending}>
                    {t('Cancel')}
                  </AlertDialogCancel>
                  <AlertDialogAction
                    type='button'
                    onClick={() => recalculateMutation.mutate()}
                    disabled={recalculateMutation.isPending}
                  >
                    {recalculateMutation.isPending ? (
                      <Spinner data-icon='inline-start' />
                    ) : (
                      <HugeiconsIcon
                        icon={ReloadIcon}
                        data-icon='inline-start'
                        strokeWidth={2}
                      />
                    )}
                    {recalculateMutation.isPending
                      ? t('Processing...')
                      : t('Confirm recalculation')}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </SettingsFormGridItem>
      </SettingsFormGrid>
    </div>
  )
}
