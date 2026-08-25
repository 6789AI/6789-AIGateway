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
import { HugeiconsIcon } from '@hugeicons/react'
import { useWatch, type FieldPath, type UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
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
  MARKETING_BANNER_ICON_NAMES,
  MARKETING_BANNER_ICON_OPTIONS,
  MarketingBannerIcon,
  getReadableTextColor,
  safeHexColor,
} from '@/features/global-banner'
import { getCountdownParts } from '@/features/global-banner/countdown'

import {
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import type { NoticeFormValues } from './notice-section'

const hexColorPattern = /^#[0-9a-fA-F]{6}$/

type BannerFieldNames = {
  enabled: FieldPath<NoticeFormValues>
  content: FieldPath<NoticeFormValues>
  backgroundColor: FieldPath<NoticeFormValues>
  textColor: FieldPath<NoticeFormValues>
  icon: FieldPath<NoticeFormValues>
  countdownEnabled: FieldPath<NoticeFormValues>
  countdownEndAt: FieldPath<NoticeFormValues>
  linkURL: FieldPath<NoticeFormValues>
  buttonText: FieldPath<NoticeFormValues>
  buttonColor: FieldPath<NoticeFormValues>
}

const globalBannerFields = {
  enabled: 'GlobalBannerEnabled',
  content: 'GlobalBannerContent',
  backgroundColor: 'GlobalBannerBackgroundColor',
  textColor: 'GlobalBannerTextColor',
  icon: 'GlobalBannerIcon',
  countdownEnabled: 'GlobalBannerCountdownEnabled',
  countdownEndAt: 'GlobalBannerCountdownEndAt',
  linkURL: 'GlobalBannerLinkURL',
  buttonText: 'GlobalBannerButtonText',
  buttonColor: 'GlobalBannerButtonColor',
} as const satisfies BannerFieldNames

const marketingBannerFields = {
  enabled: 'MarketingBannerEnabled',
  content: 'MarketingBannerContent',
  backgroundColor: 'MarketingBannerBackgroundColor',
  textColor: 'MarketingBannerTextColor',
  icon: 'MarketingBannerIcon',
  countdownEnabled: 'MarketingBannerCountdownEnabled',
  countdownEndAt: 'MarketingBannerCountdownEndAt',
  linkURL: 'MarketingBannerLinkURL',
  buttonText: 'MarketingBannerButtonText',
  buttonColor: 'MarketingBannerButtonColor',
} as const satisfies BannerFieldNames

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

type BannerFieldsProps = {
  form: UseFormReturn<NoticeFormValues>
  kind: 'global' | 'free-model'
}

export function BannerFields(props: BannerFieldsProps) {
  const { t } = useTranslation()
  const isGlobal = props.kind === 'global'
  const fields = isGlobal ? globalBannerFields : marketingBannerFields
  const watchedValues = useWatch({
    control: props.form.control,
    name: [
      fields.enabled,
      fields.content,
      fields.backgroundColor,
      fields.textColor,
      fields.icon,
      fields.buttonText,
      fields.buttonColor,
      fields.countdownEnabled,
      fields.countdownEndAt,
      fields.linkURL,
    ],
  })
  const enabled = Boolean(watchedValues[0])
  const content = String(watchedValues[1] ?? '')
  const backgroundColor = String(watchedValues[2] ?? '')
  const textColor = String(watchedValues[3] ?? '')
  const rawIcon = String(watchedValues[4] ?? '')
  const matchedIcon = MARKETING_BANNER_ICON_NAMES.find(
    (iconName) => iconName === rawIcon
  )
  const customIcon = matchedIcon ? '' : rawIcon
  const customIconLength = [...customIcon].length
  const buttonText = String(watchedValues[5] ?? '')
  const buttonColor = String(watchedValues[6] ?? '')
  const countdownEnabled = Boolean(watchedValues[7])
  const countdownEndAt = Number(watchedValues[8]) || 0
  const linkURL = String(watchedValues[9] ?? '')
  const fallbackBackgroundColor = isGlobal ? '#0EA5E9' : '#A3E635'
  const fallbackTextColor = isGlobal ? '#082F49' : '#1A2E05'
  const fallbackButtonColor = '#FFFFFF'
  const previewBackgroundColor = hexColorPattern.test(backgroundColor)
    ? backgroundColor
    : fallbackBackgroundColor
  const previewTextColor = hexColorPattern.test(textColor)
    ? textColor
    : fallbackTextColor
  const previewButtonColor = safeHexColor(buttonColor, fallbackButtonColor)
  const remainingSeconds = Math.max(
    0,
    Math.ceil(countdownEndAt - Date.now() / 1000)
  )
  const countdown = getCountdownParts(remainingSeconds)
  const countdownText = `${countdown.days}:${countdown.hours}:${countdown.minutes}:${countdown.seconds}`
  const sectionTitle = isGlobal
    ? t('Global announcement banner')
    : t('Limited-time model offer')
  const sectionDescription = isGlobal
    ? t('Display a fixed announcement at the top of every page.')
    : t(
        'Customize the icon, colors, and link used by scheduled model promotions. Content and countdown are generated automatically.'
      )
  const previewContent = isGlobal
    ? content.trim() || t('Planned maintenance on Friday at 22:00 UTC...')
    : t('{{model}} is {{percent}}% off', {
        model: 'Model A',
        percent: '20',
      })

  return (
    <section className={isGlobal ? 'space-y-6' : 'space-y-6 border-t pt-6'}>
      <div className='flex flex-col gap-1'>
        <h3 className='text-sm font-semibold'>{sectionTitle}</h3>
        <p className='text-muted-foreground text-xs'>{sectionDescription}</p>
      </div>

      {isGlobal && (
        <FormField
          control={props.form.control}
          name={fields.enabled}
          render={({ field }) => (
            <SettingsSwitchItem>
              <SettingsSwitchContent>
                <FormLabel>{t('Show global announcement banner')}</FormLabel>
                <FormDescription>
                  {t('The banner cannot be dismissed by users.')}
                </FormDescription>
              </SettingsSwitchContent>
              <FormControl>
                <Switch
                  checked={Boolean(field.value)}
                  onCheckedChange={field.onChange}
                />
              </FormControl>
            </SettingsSwitchItem>
          )}
        />
      )}

      {isGlobal && (
        <FormField
          control={props.form.control}
          name={fields.content}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Banner content')}</FormLabel>
              <FormControl>
                <Input
                  name={field.name}
                  ref={field.ref}
                  onBlur={field.onBlur}
                  onChange={field.onChange}
                  value={String(field.value ?? '')}
                  maxLength={300}
                  placeholder={t(
                    'Planned maintenance on Friday at 22:00 UTC...'
                  )}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      )}

      <FormField
        control={props.form.control}
        name={fields.icon}
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Banner icon')}</FormLabel>
            <FormControl>
              <ToggleGroup
                value={matchedIcon ? [matchedIcon] : []}
                onValueChange={(values) => {
                  field.onChange(values[0] ?? '')
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
            <div className='relative max-w-xl'>
              <Input
                name={field.name}
                ref={field.ref}
                onBlur={field.onBlur}
                onChange={(event) => {
                  field.onChange([...event.target.value].slice(0, 100).join(''))
                }}
                value={customIcon}
                aria-label={t('Custom characters or emoji')}
                placeholder={t('Enter custom characters or emoji')}
                className='pr-16'
              />
              <span className='text-muted-foreground pointer-events-none absolute inset-y-0 right-3 flex items-center text-xs tabular-nums'>
                {customIconLength}/100
              </span>
            </div>
            <FormDescription>
              {t(
                'Custom characters replace the selected preset icon and are displayed as plain text.'
              )}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      {isGlobal && (
        <FormField
          control={props.form.control}
          name={fields.countdownEnabled}
          render={({ field }) => (
            <SettingsSwitchItem>
              <SettingsSwitchContent>
                <FormLabel>{t('Enable countdown')}</FormLabel>
                <FormDescription>
                  {t(
                    'Hide the global announcement banner when the countdown ends.'
                  )}
                </FormDescription>
              </SettingsSwitchContent>
              <FormControl>
                <Switch
                  checked={Boolean(field.value)}
                  onCheckedChange={field.onChange}
                />
              </FormControl>
            </SettingsSwitchItem>
          )}
        />
      )}

      {isGlobal && countdownEnabled && (
        <FormField
          control={props.form.control}
          name={fields.countdownEndAt}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Countdown end time')}</FormLabel>
              <FormControl>
                <Input
                  type='datetime-local'
                  value={formatDateTimeLocal(Number(field.value) || 0)}
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
        name={fields.linkURL}
        render={({ field }) => (
          <FormItem>
            <FormLabel>{t('Jump link')}</FormLabel>
            <FormControl>
              <Input
                name={field.name}
                ref={field.ref}
                onBlur={field.onBlur}
                onChange={field.onChange}
                value={String(field.value ?? '')}
                type='text'
                inputMode='url'
                maxLength={2048}
                placeholder='https://example.com/promotion'
              />
            </FormControl>
            <FormDescription>
              {isGlobal
                ? t('Leave blank to disable banner navigation.')
                : t('Leave blank to open Model Square.')}
            </FormDescription>
            <FormMessage />
          </FormItem>
        )}
      />

      <div className='grid gap-4 sm:grid-cols-2'>
        <FormField
          control={props.form.control}
          name={fields.buttonText}
          render={({ field }) => (
            <FormItem>
              <FormLabel>{t('Button text')}</FormLabel>
              <FormControl>
                <Input
                  name={field.name}
                  ref={field.ref}
                  onBlur={field.onBlur}
                  onChange={field.onChange}
                  value={String(field.value ?? '')}
                  maxLength={50}
                  placeholder={t('Click to view')}
                />
              </FormControl>
              <FormDescription>
                {t('Leave blank to use the default button text.')}
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <FormField
          control={props.form.control}
          name={fields.buttonColor}
          render={({ field }) => {
            const value = String(field.value ?? '')
            return (
              <FormItem>
                <FormLabel>{t('Button color')}</FormLabel>
                <div className='flex items-center gap-2'>
                  <input
                    type='color'
                    value={safeHexColor(value, fallbackButtonColor)}
                    onChange={(event) => field.onChange(event.target.value)}
                    className='border-input h-8 w-10 shrink-0 cursor-pointer rounded-md border bg-transparent p-0.5'
                    aria-label={t('Button color')}
                  />
                  <FormControl>
                    <Input
                      name={field.name}
                      ref={field.ref}
                      onBlur={field.onBlur}
                      onChange={field.onChange}
                      value={value}
                      maxLength={7}
                    />
                  </FormControl>
                </div>
                <FormMessage />
              </FormItem>
            )
          }}
        />
      </div>

      <div className='grid gap-4 sm:grid-cols-2'>
        <FormField
          control={props.form.control}
          name={fields.backgroundColor}
          render={({ field }) => {
            const value = String(field.value ?? '')
            return (
              <FormItem>
                <FormLabel>{t('Background color')}</FormLabel>
                <div className='flex items-center gap-2'>
                  <input
                    type='color'
                    value={
                      hexColorPattern.test(value)
                        ? value
                        : fallbackBackgroundColor
                    }
                    onChange={(event) => field.onChange(event.target.value)}
                    className='border-input h-8 w-10 shrink-0 cursor-pointer rounded-md border bg-transparent p-0.5'
                    aria-label={t('Background color')}
                  />
                  <FormControl>
                    <Input
                      name={field.name}
                      ref={field.ref}
                      onBlur={field.onBlur}
                      onChange={field.onChange}
                      value={value}
                      maxLength={7}
                    />
                  </FormControl>
                </div>
                <FormMessage />
              </FormItem>
            )
          }}
        />

        <FormField
          control={props.form.control}
          name={fields.textColor}
          render={({ field }) => {
            const value = String(field.value ?? '')
            return (
              <FormItem>
                <FormLabel>{t('Text color')}</FormLabel>
                <div className='flex items-center gap-2'>
                  <input
                    type='color'
                    value={
                      hexColorPattern.test(value) ? value : fallbackTextColor
                    }
                    onChange={(event) => field.onChange(event.target.value)}
                    className='border-input h-8 w-10 shrink-0 cursor-pointer rounded-md border bg-transparent p-0.5'
                    aria-label={t('Text color')}
                  />
                  <FormControl>
                    <Input
                      name={field.name}
                      ref={field.ref}
                      onBlur={field.onBlur}
                      onChange={field.onChange}
                      value={value}
                      maxLength={7}
                    />
                  </FormControl>
                </div>
                <FormMessage />
              </FormItem>
            )
          }}
        />
      </div>

      <div className='flex flex-col gap-2'>
        <FormLabel>{t('Banner preview')}</FormLabel>
        <div
          className='flex h-10 min-w-0 items-center justify-center gap-2 overflow-hidden px-3'
          style={{
            backgroundColor: previewBackgroundColor,
            color: previewTextColor,
            opacity: isGlobal && !enabled ? 0.55 : 1,
          }}
          aria-label={t('Banner preview')}
        >
          <MarketingBannerIcon name={rawIcon} className='size-4 shrink-0' />
          <span className='truncate text-xs font-semibold'>
            {previewContent}
          </span>
          {(isGlobal ? countdownEnabled : true) && (
            <span className='shrink-0 rounded bg-white/90 px-2 py-1 font-mono text-xs font-semibold text-black/80 tabular-nums'>
              {isGlobal ? countdownText : '00:01:23:45'}
            </span>
          )}
          {(!isGlobal || linkURL.trim()) && (
            <Button
              type='button'
              size='xs'
              className='max-w-20 px-2 sm:max-w-36'
              style={{
                backgroundColor: previewButtonColor,
                color: getReadableTextColor(previewButtonColor),
              }}
            >
              <span className='truncate'>
                {buttonText.trim() || t('Click to view')}
              </span>
            </Button>
          )}
        </div>
      </div>
    </section>
  )
}
