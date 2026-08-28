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
import { Separator } from '@/components/ui/separator'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { BTMailSettingsFields } from './bt-mail-settings-fields'
import {
  buildEmailOptionUpdates,
  createEmailSchema,
  type EmailFormValues,
} from './email-settings-schema'
import { SMTPSettingsFields } from './smtp-settings-fields'

type EmailSettingsSectionProps = {
  defaultValues: EmailFormValues
  btMailPasswordConfigured: boolean
}

export function EmailSettingsSection(props: EmailSettingsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const emailSchema = createEmailSchema(t, props.btMailPasswordConfigured)
  const form = useForm<EmailFormValues>({
    resolver: zodResolver(emailSchema),
    defaultValues: props.defaultValues,
    shouldUnregister: false,
  })

  useResetForm(form, props.defaultValues)
  const provider = form.watch('EmailSenderProvider')

  const onSubmit = async (values: EmailFormValues) => {
    const updates = buildEmailOptionUpdates(values, props.defaultValues)
    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
  }

  return (
    <SettingsSection title={t('Sending settings')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save sending settings'
          />

          <FormField
            control={form.control}
            name='EmailSenderProvider'
            render={({ field }) => (
              <FormItem>
                <FormLabel id='email-sender-provider-label'>
                  {t('Sending method')}
                </FormLabel>
                <FormControl>
                  <ToggleGroup
                    aria-labelledby='email-sender-provider-label'
                    value={[field.value]}
                    onValueChange={(values) => {
                      const nextProvider = values[0]
                      if (nextProvider === 'smtp' || nextProvider === 'bt_mail') {
                        field.onChange(nextProvider)
                      }
                    }}
                    variant='outline'
                    className='grid w-full grid-cols-2'
                  >
                    <ToggleGroupItem value='smtp' className='w-full'>
                      SMTP
                    </ToggleGroupItem>
                    <ToggleGroupItem value='bt_mail' className='w-full'>
                      {t('BT Mail API')}
                    </ToggleGroupItem>
                  </ToggleGroup>
                </FormControl>
                <FormDescription>
                  {t(
                    'Choose how system emails are delivered; each method keeps its own configuration'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <Separator />

          {provider === 'smtp' ? (
            <SMTPSettingsFields form={form} />
          ) : (
            <BTMailSettingsFields
              form={form}
              passwordConfigured={props.btMailPasswordConfigured}
            />
          )}
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
