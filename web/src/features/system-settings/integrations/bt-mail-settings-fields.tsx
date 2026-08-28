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

import type { EmailFormValues } from './email-settings-schema'
import { SenderIdentityFields } from './sender-identity-fields'

type BTMailSettingsFieldsProps = {
  form: UseFormReturn<EmailFormValues>
  passwordConfigured: boolean
}

export function BTMailSettingsFields(props: BTMailSettingsFieldsProps) {
  const { t } = useTranslation()

  return (
    <div className='space-y-6'>
      <FormField
        control={props.form.control}
        name='BTMailAPIURL'
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('BT Mail API URL')}</FormLabel>
            <FormControl>
              <Input
                autoComplete='url'
                type='url'
                placeholder='https://panel.example.com:8888/mail_sys/send_mail_http.json'
                {...field}
              />
            </FormControl>
            <FormDescription>
              {t('Use the complete send_mail_http URL; HTTPS is recommended')}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <SenderIdentityFields form={props.form} provider='bt_mail' />

      <FormField
        control={props.form.control}
        name='BTMailPassword'
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Mailbox password')}</FormLabel>
            <FormControl>
              <Input
                autoComplete='new-password'
                type='password'
                placeholder={t('Enter a new mailbox password to update')}
                {...field}
              />
            </FormControl>
            <FormDescription>
              {props.passwordConfigured
                ? t('A mailbox password is configured. Leave blank to keep it.')
                : t('Use the mailbox password, not the BT Panel API key')}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />
    </div>
  )
}
