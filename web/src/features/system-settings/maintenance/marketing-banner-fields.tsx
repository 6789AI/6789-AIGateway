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
import { ArrowRight01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useWatch, type UseFormReturn } from 'react-hook-form'
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
import { Switch } from '@/components/ui/switch'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  MARKETING_BANNER_ICON_OPTIONS,
  MarketingBannerIcon,
} from '@/features/global-banner'
import { getCountdownParts } from '@/features/global-banner/countdown'

import {
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import type { NoticeFormValues } from './notice-section'

const hexColorPattern = /^#[0-9a-fA-F]{6}$/

function formatDateTimeLocal(timestamp: number): string {
  if (!timestamp) return ''
  const date = new Date(timestamp * 1000)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

function parseDateTimeLocal(value: string): number {
  if (!value) return 0
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) ? Math.floor(timestamp / 1000) : 0
}

type MarketingBannerFieldsProps = {
  form: UseFormReturn<NoticeFormValues>
}

export function MarketingBannerFields(props: MarketingBannerFieldsProps) {
  const { t } = useTranslation()
  const [
    marketingEnabled,
    marketingContent,
    backgroundColor,
    textColor,
    icon,
    countdownEnabled,
    countdownEndAt,
    linkURL,
  ] = useWatch({
    control: props.form.control,
    name: [
      'MarketingBannerEnabled',
      'MarketingBannerContent',
      'MarketingBannerBackgroundColor',
      'MarketingBannerTextColor',
      'MarketingBannerIcon',
      'MarketingBannerCountdownEnabled',
      'MarketingBannerCountdownEndAt',
      'MarketingBannerLinkURL',
    ],
  })
  const previewBackgroundColor = hexColorPattern.test(backgroundColor)
    ? backgroundColor
    : '#A3E635'
  const previewTextColor = hexColorPattern.test(textColor)
    ? textColor
    : '#1A2E05'
  const remainingSeconds = Math.max(
    0,
    Math.ceil(countdownEndAt - Date.now() / 1000)
  )
  const countdown = getCountdownParts(remainingSeconds)
  const countdownText = `${countdown.days}:${countdown.hours}:${countdown.minutes}:${countdown.seconds}`

  return (
    <>
      <div className='flex flex-col gap-1 border-t pt-6'>
        <h3 className='text-sm font-semibold'>{t('Marketing banner')}</h3>
        <p className='text-muted-foreground text-xs'>
          {t('Display a fixed marketing message at the top of every page.')}
        </p>
      </div>

      <FormField
        control={props.form.control}
        name='MarketingBannerEnabled'
        render={({ field }) => (
          <SettingsSwitchItem>
            <SettingsSwitchContent>
              <FormLabel>{t('Show marketing banner')}</FormLabel>
              <FormDescription>
                {t('The banner cannot be dismissed by users.')}
              </FormDescription>
            </SettingsSwitchContent>
            <FormControl>
              <Switch checked={field.value} onCheckedChange={field.onChange} />
            </FormControl>
          </SettingsSwitchItem>
        )}
      />

      <FormField
        control={props.form.control}
        name='MarketingBannerContent'
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Banner content')}</FormLabel>
            <FormControl>
              <Input
                maxLength={300}
                placeholder={t('Limited-time offer is now available')}
                {...field}
              />
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={props.form.control}
        name='MarketingBannerIcon'
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Banner icon')}</FormLabel>
            <FormControl>
              <ToggleGroup
                value={[field.value]}
                onValueChange={(values) => {
                  const next = values.find((value) => value !== field.value)
                  if (next) field.onChange(next)
                }}
                variant='outline'
                spacing={2}
                className='flex-wrap justify-start'
                aria-label={t('Banner icon')}
              >
                {MARKETING_BANNER_ICON_OPTIONS.map((option) => (
                  <Tooltip key={option.value}>
                    <TooltipTrigger
                      render={
                        <ToggleGroupItem
                          value={option.value}
                          aria-label={t(option.labelKey)}
                        />
                      }
                    >
                      <HugeiconsIcon
                        icon={option.icon}
                        strokeWidth={2}
                        aria-hidden='true'
                      />
                    </TooltipTrigger>
                    <TooltipContent>{t(option.labelKey)}</TooltipContent>
                  </Tooltip>
                ))}
              </ToggleGroup>
            </FormControl>
            <FormMessage />
          </FormItem>
        )}
      />

      <FormField
        control={props.form.control}
        name='MarketingBannerCountdownEnabled'
        render={({ field }) => (
          <SettingsSwitchItem>
            <SettingsSwitchContent>
              <FormLabel>{t('Enable countdown')}</FormLabel>
              <FormDescription>
                {t('Hide the marketing banner when the countdown ends.')}
              </FormDescription>
            </SettingsSwitchContent>
            <FormControl>
              <Switch checked={field.value} onCheckedChange={field.onChange} />
            </FormControl>
          </SettingsSwitchItem>
        )}
      />

      {countdownEnabled && (
        <FormField
          control={props.form.control}
          name='MarketingBannerCountdownEndAt'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Countdown end time')}</FormLabel>
              <FormControl>
                <Input
                  type='datetime-local'
                  value={formatDateTimeLocal(field.value)}
                  onChange={(event) =>
                    field.onChange(parseDateTimeLocal(event.target.value))
                  }
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      )}

      <FormField
        control={props.form.control}
        name='MarketingBannerLinkURL'
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Jump link')}</FormLabel>
            <FormControl>
              <Input
                type='text'
                inputMode='url'
                maxLength={2048}
                placeholder='https://example.com/promotion'
                {...field}
              />
            </FormControl>
            <FormDescription>
              {t('Leave blank to disable banner navigation.')}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <div className='grid gap-4 sm:grid-cols-2'>
        <FormField
          control={props.form.control}
          name='MarketingBannerBackgroundColor'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Background color')}</FormLabel>
              <div className='flex items-center gap-2'>
                <input
                  type='color'
                  value={
                    hexColorPattern.test(field.value) ? field.value : '#A3E635'
                  }
                  onChange={field.onChange}
                  className='border-input h-8 w-10 shrink-0 cursor-pointer rounded-md border bg-transparent p-0.5'
                  aria-label={t('Background color')}
                />
                <FormControl>
                  <Input maxLength={7} {...field} />
                </FormControl>
              </div>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={props.form.control}
          name='MarketingBannerTextColor'
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Text color')}</FormLabel>
              <div className='flex items-center gap-2'>
                <input
                  type='color'
                  value={
                    hexColorPattern.test(field.value) ? field.value : '#1A2E05'
                  }
                  onChange={field.onChange}
                  className='border-input h-8 w-10 shrink-0 cursor-pointer rounded-md border bg-transparent p-0.5'
                  aria-label={t('Text color')}
                />
                <FormControl>
                  <Input maxLength={7} {...field} />
                </FormControl>
              </div>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>

      <div className='flex flex-col gap-2'>
        <FormLabel>{t('Banner preview')}</FormLabel>
        <div
          className='flex h-10 min-w-0 items-center justify-center gap-2 overflow-hidden px-3'
          style={{
            backgroundColor: previewBackgroundColor,
            color: previewTextColor,
            opacity: marketingEnabled ? 1 : 0.55,
          }}
          aria-label={t('Banner preview')}
        >
          <MarketingBannerIcon name={icon} className='size-4 shrink-0' />
          <span className='truncate text-xs font-semibold'>
            {marketingContent.trim() ||
              t('Limited-time offer is now available')}
          </span>
          {countdownEnabled && (
            <span className='shrink-0 rounded bg-white/90 px-2 py-1 font-mono text-xs font-semibold text-black/80 tabular-nums'>
              {countdownText}
            </span>
          )}
          {linkURL.trim() && (
            <HugeiconsIcon
              icon={ArrowRight01Icon}
              strokeWidth={2}
              className='size-4 shrink-0'
              aria-hidden='true'
            />
          )}
        </div>
      </div>
    </>
  )
}
