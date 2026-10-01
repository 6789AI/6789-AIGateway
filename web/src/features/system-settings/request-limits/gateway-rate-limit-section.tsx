/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'

import { SettingsForm } from '../components/settings-form-layout'
import { useSuppressSettingsSectionHeader } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  gatewayRateLimitSchema,
  parseGatewayRateLimits,
  type GatewayRateLimits,
} from './gateway-rate-limit-schema'

type RuleName = keyof GatewayRateLimits

type GatewayRateLimitSectionProps = {
  defaultValue: string
}

export function GatewayRateLimitSection(props: GatewayRateLimitSectionProps) {
  const { t } = useTranslation()
  const suppressHeader = useSuppressSettingsSectionHeader()
  const updateOption = useUpdateOption()
  const form = useForm<GatewayRateLimits>({
    resolver: zodResolver(gatewayRateLimitSchema),
    defaultValues: parseGatewayRateLimits(props.defaultValue),
  })

  useEffect(() => {
    form.reset(parseGatewayRateLimits(props.defaultValue))
  }, [form, props.defaultValue])

  const rules: Array<{ key: RuleName; label: string; description: string }> = [
    {
      key: 'api',
      label: t('API requests'),
      description: t('All dashboard API requests per IP address.'),
    },
    {
      key: 'web',
      label: t('Web requests'),
      description: t('Pages and static files per IP address.'),
    },
    {
      key: 'critical',
      label: t('Sensitive requests'),
      description: t('Shared limit for sensitive operations.'),
    },
    {
      key: 'login',
      label: t('Login and registration'),
      description: t(
        'When off, login and registration use the sensitive request limit.'
      ),
    },
    {
      key: 'session',
      label: t('Session refresh and logout'),
      description: t(
        'When off, refresh and logout use the sensitive request limit.'
      ),
    },
  ]

  const onSubmit = async (values: GatewayRateLimits) => {
    await updateOption.mutateAsync({
      key: 'GatewayRateLimits',
      value: JSON.stringify(values),
    })
  }

  return (
    <SettingsSection title={t('Website request limits')}>
      {suppressHeader && (
        <h3 className='text-base font-semibold'>
          {t('Website request limits')}
        </h3>
      )}
      <p className='text-muted-foreground text-sm'>
        {t(
          'Limits use the client IP recognized by the server. Users sharing a public IP share the same allowance. Configure trusted proxies on the server.'
        )}
      </p>
      <Form {...form}>
        <SettingsForm noValidate onSubmit={form.handleSubmit(onSubmit)}>
          <div className='space-y-4'>
            {rules.map((rule) => (
              <fieldset
                key={rule.key}
                className='space-y-4 rounded-lg border p-4'
              >
                <legend className='px-1 text-sm font-medium'>
                  {rule.label}
                </legend>
                <p className='text-muted-foreground text-sm'>
                  {rule.description}
                </p>
                <FormField
                  control={form.control}
                  name={`${rule.key}.enabled`}
                  render={({ field }) => (
                    <FormItem className='flex items-center gap-3'>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                          aria-label={t('Enable {{name}} limit', {
                            name: rule.label,
                          })}
                        />
                      </FormControl>
                      <FormLabel>
                        {rule.key === 'login' || rule.key === 'session'
                          ? t('Use separate limit')
                          : t('Enable limit')}
                      </FormLabel>
                    </FormItem>
                  )}
                />
                <div className='grid gap-4 md:grid-cols-2'>
                  <FormField
                    control={form.control}
                    name={`${rule.key}.limit`}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Maximum requests')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            name={field.name}
                            onBlur={field.onBlur}
                            ref={field.ref}
                            min={1}
                            max={1000000}
                            step={1}
                            value={field.value}
                            onChange={(event) =>
                              field.onChange(Number(event.target.value))
                            }
                            aria-invalid={Boolean(
                              form.formState.errors[rule.key]?.limit
                            )}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name={`${rule.key}.window_seconds`}
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('Window (seconds)')}</FormLabel>
                        <FormControl>
                          <Input
                            type='number'
                            name={field.name}
                            onBlur={field.onBlur}
                            ref={field.ref}
                            min={1}
                            max={1200}
                            step={1}
                            value={field.value}
                            onChange={(event) =>
                              field.onChange(Number(event.target.value))
                            }
                            aria-invalid={Boolean(
                              form.formState.errors[rule.key]?.window_seconds
                            )}
                          />
                        </FormControl>
                        <FormDescription>
                          {t('Up to 20 minutes.')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </fieldset>
            ))}
          </div>
          <div className='flex justify-end'>
            <Button type='submit' disabled={updateOption.isPending}>
              {t('Save website request limits')}
            </Button>
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
