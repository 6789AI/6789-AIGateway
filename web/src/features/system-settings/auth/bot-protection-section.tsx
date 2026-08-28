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
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Switch } from '@/components/ui/switch'
import { cn } from '@/lib/utils'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { BotProtectionProviderFields } from './bot-protection-provider-fields'
import {
  createBotProtectionSchema,
  type BotProtectionFormValues,
  type BotProtectionProvider,
  type SecretConfigured,
} from './bot-protection-schema'

type BotProtectionSectionProps = {
  defaultValues: BotProtectionFormValues
  secretConfigured: SecretConfigured
}

const providerOptions: Array<{
  value: BotProtectionProvider
  labelKey: string
}> = [
  { value: 'turnstile', labelKey: 'Turnstile' },
  { value: 'recaptcha', labelKey: 'reCAPTCHA' },
  { value: 'geetest_v4', labelKey: 'GeeTest V4' },
  { value: 'cap', labelKey: 'Cap' },
]

const scopeOptions: Array<{
  name:
    | 'BotProtectionLoginEnabled'
    | 'BotProtectionRegisterEnabled'
    | 'BotProtectionEmailVerificationEnabled'
    | 'BotProtectionPasswordResetEnabled'
    | 'BotProtectionCheckinEnabled'
  labelKey: string
  descriptionKey: string
}> = [
  {
    name: 'BotProtectionLoginEnabled',
    labelKey: 'Login',
    descriptionKey: 'Require human verification for password login',
  },
  {
    name: 'BotProtectionRegisterEnabled',
    labelKey: 'Registration',
    descriptionKey: 'Require human verification when creating an account',
  },
  {
    name: 'BotProtectionEmailVerificationEnabled',
    labelKey: 'Email verification code',
    descriptionKey: 'Require human verification before sending an email code',
  },
  {
    name: 'BotProtectionPasswordResetEnabled',
    labelKey: 'Password reset',
    descriptionKey: 'Require human verification before sending a reset email',
  },
  {
    name: 'BotProtectionCheckinEnabled',
    labelKey: 'Daily check-in',
    descriptionKey: 'Require human verification before daily check-in',
  },
]

export function BotProtectionSection(props: BotProtectionSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const schema = useMemo(
    () => createBotProtectionSchema(t, props.secretConfigured),
    [props.secretConfigured, t]
  )
  const form = useForm<BotProtectionFormValues>({
    resolver: zodResolver(schema),
    defaultValues: props.defaultValues,
  })
  const provider = form.watch('BotProtectionProvider')

  useEffect(() => {
    form.reset(props.defaultValues)
  }, [form, props.defaultValues])

  const onSubmit = async (data: BotProtectionFormValues) => {
    const updates = Object.entries(data).filter(
      ([key, value]) =>
        value !== props.defaultValues[key as keyof BotProtectionFormValues]
    )
    const enabledUpdate = updates.find(
      ([key]) => key === 'TurnstileCheckEnabled'
    )
    const remaining = updates.filter(
      ([key]) =>
        key !== 'TurnstileCheckEnabled' && key !== 'BotProtectionProvider'
    )
    const providerUpdate = updates.find(
      ([key]) => key === 'BotProtectionProvider'
    )
    const orderedUpdates =
      enabledUpdate?.[1] === false
        ? [
            enabledUpdate,
            ...remaining,
            ...(providerUpdate ? [providerUpdate] : []),
          ]
        : [
            ...remaining,
            ...(providerUpdate ? [providerUpdate] : []),
            ...(enabledUpdate ? [enabledUpdate] : []),
          ]

    for (const [key, value] of orderedUpdates) {
      await updateOption.mutateAsync({ key, value: value ?? '' })
    }
  }

  let providerDescription = t('Choose the service used for human verification')
  if (provider === 'recaptcha') {
    providerDescription = t(
      'reCAPTCHA loads and verifies through recaptcha.net'
    )
  } else if (provider === 'cap') {
    providerDescription = t('Cap connects only to your self-hosted instance')
  }

  return (
    <SettingsSection title={t('Bot Protection')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />

          <FormField
            control={form.control}
            name='TurnstileCheckEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable bot protection')}</FormLabel>
                  <FormDescription>
                    {t('Use the selected provider on enabled user actions')}
                  </FormDescription>
                  <FormMessage />
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='BotProtectionProvider'
            render={({ field }) => (
              <FormItem>
                <FormLabel id='bot-protection-provider-label'>
                  {t('Provider')}
                </FormLabel>
                <FormControl>
                  <div
                    role='radiogroup'
                    aria-labelledby='bot-protection-provider-label'
                    className='bg-muted/60 grid grid-cols-2 gap-1 rounded-lg border p-1 sm:grid-cols-4'
                  >
                    {providerOptions.map((option) => {
                      const selected = field.value === option.value
                      return (
                        <button
                          key={option.value}
                          type='button'
                          role='radio'
                          aria-checked={selected}
                          onClick={() => field.onChange(option.value)}
                          className={cn(
                            'focus-visible:ring-ring min-h-9 rounded-md px-3 py-2 text-sm font-medium transition-colors focus-visible:ring-2 focus-visible:outline-none',
                            selected
                              ? 'bg-background text-foreground shadow-sm'
                              : 'text-muted-foreground hover:text-foreground'
                          )}
                        >
                          {t(option.labelKey)}
                        </button>
                      )
                    })}
                  </div>
                </FormControl>
                <FormDescription>{providerDescription}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className='space-y-1 border-t pt-5'>
            <div className='pb-2'>
              <h3 className='text-sm font-medium'>{t('Protected features')}</h3>
              <p className='text-muted-foreground text-sm'>
                {t('Select which actions require human verification')}
              </p>
            </div>
            {scopeOptions.map((option) => (
              <FormField
                key={option.name}
                control={form.control}
                name={option.name}
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{t(option.labelKey)}</FormLabel>
                      <FormDescription>
                        {t(option.descriptionKey)}
                      </FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                  </SettingsSwitchItem>
                )}
              />
            ))}
          </div>

          <BotProtectionProviderFields
            form={form}
            provider={provider}
            secretConfigured={props.secretConfigured}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
