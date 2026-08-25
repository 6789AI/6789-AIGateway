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

import { Form } from '@/components/ui/form'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { BannerFields } from './marketing-banner-fields'

const hexColorPattern = /^#[0-9a-fA-F]{6}$/

export type NoticeFormValues = {
  Notice: string
  GlobalBannerEnabled: boolean
  GlobalBannerContent: string
  GlobalBannerBackgroundColor: string
  GlobalBannerTextColor: string
  GlobalBannerIcon: string
  GlobalBannerCountdownEnabled: boolean
  GlobalBannerCountdownEndAt: number
  GlobalBannerLinkURL: string
  GlobalBannerButtonText: string
  GlobalBannerButtonColor: string
  MarketingBannerEnabled: boolean
  MarketingBannerContent: string
  MarketingBannerBackgroundColor: string
  MarketingBannerTextColor: string
  MarketingBannerIcon: string
  MarketingBannerCountdownEnabled: boolean
  MarketingBannerCountdownEndAt: number
  MarketingBannerLinkURL: string
  MarketingBannerButtonText: string
  MarketingBannerButtonColor: string
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
          GlobalBannerEnabled: z.boolean(),
          GlobalBannerContent: z.string().max(300),
          GlobalBannerBackgroundColor: z
            .string()
            .regex(hexColorPattern, t('Colors must use a 6-digit hex value')),
          GlobalBannerTextColor: z
            .string()
            .regex(hexColorPattern, t('Colors must use a 6-digit hex value')),
          GlobalBannerIcon: z
            .string()
            .refine(
              (value) => [...value.trim()].length <= 100,
              t('Custom banner characters must not exceed 100 characters')
            ),
          GlobalBannerCountdownEnabled: z.boolean(),
          GlobalBannerCountdownEndAt: z.number().int().nonnegative(),
          GlobalBannerLinkURL: z.string().trim().max(2048),
          GlobalBannerButtonText: z.string().trim().max(50),
          GlobalBannerButtonColor: z
            .string()
            .regex(hexColorPattern, t('Colors must use a 6-digit hex value')),
          MarketingBannerEnabled: z.boolean(),
          MarketingBannerContent: z.string().max(300),
          MarketingBannerBackgroundColor: z
            .string()
            .regex(hexColorPattern, t('Colors must use a 6-digit hex value')),
          MarketingBannerTextColor: z
            .string()
            .regex(hexColorPattern, t('Colors must use a 6-digit hex value')),
          MarketingBannerIcon: z
            .string()
            .refine(
              (value) => [...value.trim()].length <= 100,
              t('Custom banner characters must not exceed 100 characters')
            ),
          MarketingBannerCountdownEnabled: z.boolean(),
          MarketingBannerCountdownEndAt: z.number().int().nonnegative(),
          MarketingBannerLinkURL: z.string().trim().max(2048),
          MarketingBannerButtonText: z.string().trim().max(50),
          MarketingBannerButtonColor: z
            .string()
            .regex(hexColorPattern, t('Colors must use a 6-digit hex value')),
        })
        .superRefine((values, context) => {
          if (
            values.GlobalBannerEnabled &&
            values.GlobalBannerContent.trim().length === 0
          ) {
            context.addIssue({
              code: 'custom',
              path: ['GlobalBannerContent'],
              message: t(
                'Content is required when the global announcement banner is enabled'
              ),
            })
          }
          if (
            values.GlobalBannerEnabled &&
            values.GlobalBannerCountdownEnabled &&
            values.GlobalBannerCountdownEndAt <= Date.now() / 1000
          ) {
            context.addIssue({
              code: 'custom',
              path: ['GlobalBannerCountdownEndAt'],
              message: t('Countdown end time must be in the future'),
            })
          }
          const globalLink = values.GlobalBannerLinkURL.trim()
          if (globalLink) {
            let validGlobalLink = /^\/(?!\/)/.test(globalLink)
            if (!validGlobalLink) {
              try {
                const parsed = new URL(globalLink)
                validGlobalLink =
                  parsed.protocol === 'http:' || parsed.protocol === 'https:'
              } catch {
                validGlobalLink = false
              }
            }
            if (!validGlobalLink) {
              context.addIssue({
                code: 'custom',
                path: ['GlobalBannerLinkURL'],
                message: t(
                  'Enter an HTTP(S) URL or a site path starting with /'
                ),
              })
            }
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
        key: 'global_banner.content',
        value: values.GlobalBannerContent.trim(),
        previous: props.defaultValues.GlobalBannerContent,
      },
      {
        key: 'global_banner.background_color',
        value: values.GlobalBannerBackgroundColor.toUpperCase(),
        previous: props.defaultValues.GlobalBannerBackgroundColor,
      },
      {
        key: 'global_banner.text_color',
        value: values.GlobalBannerTextColor.toUpperCase(),
        previous: props.defaultValues.GlobalBannerTextColor,
      },
      {
        key: 'global_banner.icon',
        value: values.GlobalBannerIcon.trim(),
        previous: props.defaultValues.GlobalBannerIcon,
      },
      {
        key: 'global_banner.countdown_end_at',
        value: values.GlobalBannerCountdownEndAt,
        previous: props.defaultValues.GlobalBannerCountdownEndAt,
      },
      {
        key: 'global_banner.link_url',
        value: values.GlobalBannerLinkURL.trim(),
        previous: props.defaultValues.GlobalBannerLinkURL,
      },
      {
        key: 'global_banner.button_text',
        value: values.GlobalBannerButtonText.trim(),
        previous: props.defaultValues.GlobalBannerButtonText,
      },
      {
        key: 'global_banner.button_color',
        value: values.GlobalBannerButtonColor.toUpperCase(),
        previous: props.defaultValues.GlobalBannerButtonColor,
      },
      {
        key: 'global_banner.countdown_enabled',
        value: values.GlobalBannerCountdownEnabled,
        previous: props.defaultValues.GlobalBannerCountdownEnabled,
      },
      {
        key: 'global_banner.enabled',
        value: values.GlobalBannerEnabled,
        previous: props.defaultValues.GlobalBannerEnabled,
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
        value: values.MarketingBannerIcon.trim(),
        previous: props.defaultValues.MarketingBannerIcon,
      },
      {
        key: 'marketing_banner.link_url',
        value: values.MarketingBannerLinkURL.trim(),
        previous: props.defaultValues.MarketingBannerLinkURL,
      },
      {
        key: 'marketing_banner.button_text',
        value: values.MarketingBannerButtonText.trim(),
        previous: props.defaultValues.MarketingBannerButtonText,
      },
      {
        key: 'marketing_banner.button_color',
        value: values.MarketingBannerButtonColor.toUpperCase(),
        previous: props.defaultValues.MarketingBannerButtonColor,
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
    <SettingsSection title={t('Global banners')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save banners'
          />
          <BannerFields form={form} kind='global' />
          <BannerFields form={form} kind='free-model' />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
