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
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Textarea } from '@/components/ui/textarea'
import {
  MARKETING_BANNER_ICON_NAMES,
  type MarketingBannerIconName,
} from '@/features/global-banner'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { MarketingBannerFields } from './marketing-banner-fields'

const hexColorPattern = /^#[0-9a-fA-F]{6}$/

export type NoticeFormValues = {
  Notice: string
  MarketingBannerEnabled: boolean
  MarketingBannerContent: string
  MarketingBannerBackgroundColor: string
  MarketingBannerTextColor: string
  MarketingBannerIcon: MarketingBannerIconName
  MarketingBannerCountdownEnabled: boolean
  MarketingBannerCountdownEndAt: number
  MarketingBannerLinkURL: string
}

type NoticeSectionProps = {
  defaultValues: NoticeFormValues
}

export function NoticeSection(props: NoticeSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const noticeSchema = useMemo(
    () =>
      z
        .object({
          Notice: z.string(),
          MarketingBannerEnabled: z.boolean(),
          MarketingBannerContent: z.string().max(300),
          MarketingBannerBackgroundColor: z
            .string()
            .regex(hexColorPattern, t('Colors must use a 6-digit hex value')),
          MarketingBannerTextColor: z
            .string()
            .regex(hexColorPattern, t('Colors must use a 6-digit hex value')),
          MarketingBannerIcon: z.enum(MARKETING_BANNER_ICON_NAMES),
          MarketingBannerCountdownEnabled: z.boolean(),
          MarketingBannerCountdownEndAt: z.number().int().nonnegative(),
          MarketingBannerLinkURL: z.string().trim().max(2048),
        })
        .superRefine((values, context) => {
          if (
            values.MarketingBannerEnabled &&
            values.MarketingBannerContent.trim().length === 0
          ) {
            context.addIssue({
              code: 'custom',
              path: ['MarketingBannerContent'],
              message: t(
                'Content is required when the marketing banner is enabled'
              ),
            })
          }
          if (
            values.MarketingBannerEnabled &&
            values.MarketingBannerCountdownEnabled &&
            values.MarketingBannerCountdownEndAt <= Date.now() / 1000
          ) {
            context.addIssue({
              code: 'custom',
              path: ['MarketingBannerCountdownEndAt'],
              message: t('Countdown end time must be in the future'),
            })
          }
          const link = values.MarketingBannerLinkURL.trim()
          if (link) {
            let validLink = /^\/(?!\/)/.test(link)
            if (!validLink) {
              try {
                const parsed = new URL(link)
                validLink =
                  parsed.protocol === 'http:' || parsed.protocol === 'https:'
              } catch {
                validLink = false
              }
            }
            if (!validLink) {
              context.addIssue({
                code: 'custom',
                path: ['MarketingBannerLinkURL'],
                message: t(
                  'Enter an HTTP(S) URL or a site path starting with /'
                ),
              })
            }
          }
        }),
    [t]
  )
  const form = useForm<NoticeFormValues>({
    resolver: zodResolver(noticeSchema),
    defaultValues: props.defaultValues,
  })

  useEffect(() => {
    form.reset(props.defaultValues)
  }, [form, props.defaultValues])

  const onSubmit = async (values: NoticeFormValues) => {
    const updates: Array<{
      key: string
      value: string | boolean | number
      previous: string | boolean | number
    }> = [
      {
        key: 'Notice',
        value: values.Notice,
        previous: props.defaultValues.Notice,
      },
      {
        key: 'marketing_banner.content',
        value: values.MarketingBannerContent.trim(),
        previous: props.defaultValues.MarketingBannerContent,
      },
      {
        key: 'marketing_banner.background_color',
        value: values.MarketingBannerBackgroundColor.toUpperCase(),
        previous: props.defaultValues.MarketingBannerBackgroundColor,
      },
      {
        key: 'marketing_banner.text_color',
        value: values.MarketingBannerTextColor.toUpperCase(),
        previous: props.defaultValues.MarketingBannerTextColor,
      },
      {
        key: 'marketing_banner.icon',
        value: values.MarketingBannerIcon,
        previous: props.defaultValues.MarketingBannerIcon,
      },
      {
        key: 'marketing_banner.countdown_end_at',
        value: values.MarketingBannerCountdownEndAt,
        previous: props.defaultValues.MarketingBannerCountdownEndAt,
      },
      {
        key: 'marketing_banner.link_url',
        value: values.MarketingBannerLinkURL.trim(),
        previous: props.defaultValues.MarketingBannerLinkURL,
      },
      {
        key: 'marketing_banner.countdown_enabled',
        value: values.MarketingBannerCountdownEnabled,
        previous: props.defaultValues.MarketingBannerCountdownEnabled,
      },
      {
        key: 'marketing_banner.enabled',
        value: values.MarketingBannerEnabled,
        previous: props.defaultValues.MarketingBannerEnabled,
      },
    ]

    for (const update of updates) {
      if (update.value === update.previous) continue
      await updateOption.mutateAsync({
        key: update.key,
        value: update.value,
      })
    }
  }

  return (
    <SettingsSection title={t('System Notice')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save notice and banner'
          />
          <FormField
            control={form.control}
            name='Notice'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Announcement content')}</FormLabel>
                <FormControl>
                  <Textarea
                    rows={5}
                    placeholder={t(
                      'Planned maintenance on Friday at 22:00 UTC...'
                    )}
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <MarketingBannerFields form={form} />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
