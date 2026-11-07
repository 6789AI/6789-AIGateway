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
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { useSidebarData } from '@/hooks/use-sidebar-data'
import {
  NAV_COLOR_PATTERN,
  type SidebarNavColors,
} from '@/lib/navigation-colors'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { NavigationColorInput } from './navigation-color-input'

const colorsSchema = z.record(
  z.string(),
  z
    .string()
    .refine((value) => !value || NAV_COLOR_PATTERN.test(value), {
      message: 'Invalid color',
    })
    .optional()
)

type SidebarColorFormValues = z.infer<typeof colorsSchema>

type SidebarColorsSectionProps = {
  config: SidebarNavColors
  initialSerialized: string
}

export function SidebarColorsSection(props: SidebarColorsSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const groups = useSidebarData().navGroups
  const form = useForm<SidebarColorFormValues>({
    resolver: zodResolver(colorsSchema),
    defaultValues: props.config,
  })

  useEffect(() => {
    form.reset(props.config)
  }, [form, props.config])

  const onSubmit = async (values: SidebarColorFormValues) => {
    const colors = { ...props.config }
    for (const group of groups) {
      for (const item of group.items) {
        const key = item.url ?? item.type
        if (!key) continue
        if (values[key]) colors[key] = values[key]
        else delete colors[key]
      }
    }
    const serialized = JSON.stringify(colors)
    if (serialized === props.initialSerialized) return

    await updateOption.mutateAsync({
      key: 'SidebarNavColors',
      value: serialized,
    })
  }

  return (
    <SettingsSection title={t('Sidebar colors')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            onReset={() => form.reset({})}
            isSaving={updateOption.isPending}
            resetLabel='Reset to default'
            saveLabel='Save colors'
          />
          {groups.map((group) => (
            <div key={group.id ?? group.title} className='space-y-2'>
              <h3 className='text-muted-foreground text-xs font-medium'>
                {group.title}
              </h3>
              <div className='divide-y border-y'>
                {group.items.map((item) => {
                  const key = item.url ?? item.type
                  if (!key) return null
                  const Icon = item.icon
                  return (
                    <FormField
                      key={key}
                      control={form.control}
                      name={key}
                      render={({ field }) => (
                        <FormItem className='grid min-w-0 gap-2 py-3 sm:grid-cols-[minmax(0,1fr)_minmax(12rem,16rem)] sm:items-center'>
                          <FormLabel className='flex min-w-0 items-center gap-2'>
                            {Icon && <Icon className='size-4 shrink-0' />}
                            <span className='truncate'>{item.title}</span>
                          </FormLabel>
                          <div className='min-w-0'>
                            <NavigationColorInput
                              label={`${item.title} ${t('Text color')}`}
                              value={field.value ?? ''}
                              name={field.name}
                              onChange={field.onChange}
                              onBlur={field.onBlur}
                              inputRef={field.ref}
                            />
                            <FormMessage />
                          </div>
                        </FormItem>
                      )}
                    />
                  )
                })}
              </div>
            </div>
          ))}
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
