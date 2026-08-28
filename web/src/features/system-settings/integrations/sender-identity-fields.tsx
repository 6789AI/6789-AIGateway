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

import type {
  EmailFormValues,
  EmailSenderProvider,
} from './email-settings-schema'

type SenderIdentityFieldsProps = {
  form: UseFormReturn<EmailFormValues>
  provider: EmailSenderProvider
}

export function SenderIdentityFields(props: SenderIdentityFieldsProps) {
  const { t } = useTranslation()
  const nameField =
    props.provider === 'smtp' ? 'SMTPFromName' : 'BTMailFromName'
  const emailField = props.provider === 'smtp' ? 'SMTPFrom' : 'BTMailFrom'

  return (
    <>
      <FormField
        control={props.form.control}
        name={nameField}
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Sender name')}</FormLabel>
            <FormControl>
              <Input
                autoComplete='organization'
                placeholder={t('New API')}
                {...field}
              />
            </FormControl>
            <FormDescription>
              {props.provider === 'smtp'
                ? t('Used as the display name in the email From header')
                : t(
                    'Shown at the top of email content; BT Mail only accepts the mailbox address'
                  )}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={props.form.control}
        name={emailField}
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Sender email')}</FormLabel>
            <FormControl>
              <Input
                autoComplete='email'
                type='email'
                placeholder={t('noreply@example.com')}
                {...field}
              />
            </FormControl>
            <FormDescription>
              {props.provider === 'smtp'
                ? t('Address used in the From header and SMTP envelope')
                : t('Mailbox account used as mail_from by BT Mail')}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

    </>
  )
}
