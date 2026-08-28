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
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Switch } from '@/components/ui/switch'

import {
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import {
  type EmailFormValues,
  getSmtpSecurityMode,
  type SmtpSecurityMode,
} from './email-settings-schema'
import { SenderIdentityFields } from './sender-identity-fields'

type SMTPSettingsFieldsProps = {
  form: UseFormReturn<EmailFormValues>
}

export function SMTPSettingsFields(props: SMTPSettingsFieldsProps) {
  const { t } = useTranslation()
  const sslEnabled = props.form.watch('SMTPSSLEnabled')
  const startTLSEnabled = props.form.watch('SMTPStartTLSEnabled')

  return (
    <div className='space-y-6'>
      <FormField
        control={props.form.control}
        name='SMTPServer'
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('SMTP Host')}</FormLabel>
            <FormControl>
              <Input
                autoComplete='off'
                placeholder={t('smtp.example.com')}
                {...field}
              />
            </FormControl>
            <FormDescription>
              {t('Hostname or IP of your SMTP provider')}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <div className='grid gap-6 md:grid-cols-2'>
        <FormField
          control={props.form.control}
          name='SMTPPort'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Port')}</FormLabel>
              <FormControl>
                <Input
                  autoComplete='off'
                  type='number'
                  placeholder='587'
                  {...field}
                />
              </FormControl>
              <FormDescription>
                {t('Common ports include 25, 465, and 587')}
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormItem>
          <FormLabel>{t('SMTP encryption')}</FormLabel>
          <FormControl>
            <RadioGroup
              value={getSmtpSecurityMode({
                SMTPSSLEnabled: sslEnabled,
                SMTPStartTLSEnabled: startTLSEnabled,
              })}
              onValueChange={(value) => {
                const mode = value as SmtpSecurityMode
                props.form.setValue('SMTPSSLEnabled', mode === 'ssl_tls', {
                  shouldDirty: true,
                })
                props.form.setValue(
                  'SMTPStartTLSEnabled',
                  mode === 'starttls',
                  { shouldDirty: true }
                )
              }}
              className='gap-3'
            >
              <div className='flex items-center gap-2'>
                <RadioGroupItem value='none' id='smtp-security-none' />
                <Label
                  htmlFor='smtp-security-none'
                  className='cursor-pointer font-normal'
                >
                  {t('No encryption')}
                </Label>
              </div>
              <div className='flex items-center gap-2'>
                <RadioGroupItem value='ssl_tls' id='smtp-security-ssl-tls' />
                <Label
                  htmlFor='smtp-security-ssl-tls'
                  className='cursor-pointer font-normal'
                >
                  {t('SSL/TLS')}
                </Label>
              </div>
              <div className='flex items-center gap-2'>
                <RadioGroupItem
                  value='starttls'
                  id='smtp-security-starttls'
                />
                <Label
                  htmlFor='smtp-security-starttls'
                  className='cursor-pointer font-normal'
                >
                  {t('STARTTLS')}
                </Label>
              </div>
            </RadioGroup>
          </FormControl>
          <FormDescription>
            {t('Choose one SMTP transport security mode')}
          </FormDescription>
        </FormItem>

        <FormField
          control={props.form.control}
          name='SMTPInsecureSkipVerify'
          render={({ field }) => (
            <SettingsSwitchItem>
              <SettingsSwitchContent>
                <FormLabel>
                  {t('Skip SMTP TLS certificate verification')}
                </FormLabel>
                <FormDescription>
                  {t(
                    'Allow self-signed or hostname-mismatched SMTP certificates'
                  )}
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

        <FormField
          control={props.form.control}
          name='SMTPForceAuthLogin'
          render={({ field }) => (
            <SettingsSwitchItem>
              <SettingsSwitchContent>
                <FormLabel>{t('Force AUTH LOGIN')}</FormLabel>
                <FormDescription>
                  {t('Force SMTP authentication using AUTH LOGIN method')}
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
      </div>

      <FormField
        control={props.form.control}
        name='SMTPAccount'
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Username')}</FormLabel>
            <FormControl>
              <Input
                autoComplete='off'
                placeholder={t('noreply@example.com')}
                {...field}
              />
            </FormControl>
            <FormDescription>
              {t('Account used when authenticating with the SMTP server')}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <SenderIdentityFields form={props.form} provider='smtp' />

      <FormField
        control={props.form.control}
        name='SMTPToken'
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Password / Access Token')}</FormLabel>
            <FormControl>
              <Input
                autoComplete='new-password'
                type='password'
                placeholder={t('Enter new token to update')}
                {...field}
              />
            </FormControl>
            <FormDescription>
              {t('Leave blank to keep the existing credential')}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  )
}
