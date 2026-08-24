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
import {
  Add01Icon,
  Calendar03Icon,
  Clock01Icon,
  Delete02Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { toIntlLocale } from '@/i18n/languages'

import type { PriceSchedule } from './model-pricing-core'

const WEEKDAYS = [
  'Sunday',
  'Monday',
  'Tuesday',
  'Wednesday',
  'Thursday',
  'Friday',
  'Saturday',
]

const COMMON_TIMEZONES = [
  'Asia/Shanghai',
  'UTC',
  'Asia/Tokyo',
  'Europe/London',
  'America/New_York',
  'America/Los_Angeles',
]

const formatDateTimeLocal = (timestamp?: number) => {
  if (!timestamp) return ''
  const date = new Date(timestamp * 1000)
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return local.toISOString().slice(0, 16)
}

const parseDateTimeLocal = (value: string) => {
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) ? Math.floor(timestamp / 1000) : undefined
}

const formatTime = (minute?: number) => {
  const value = minute ?? 0
  return `${String(Math.floor(value / 60)).padStart(2, '0')}:${String(value % 60).padStart(2, '0')}`
}

const parseTime = (value: string) => {
  const [hour, minute] = value.split(':').map(Number)
  return hour * 60 + minute
}

const createSchedule = (allowFixedPrice: boolean): PriceSchedule => {
  const now = new Date()
  now.setSeconds(0, 0)
  const end = new Date(now)
  end.setDate(end.getDate() + 1)
  return {
    id:
      globalThis.crypto?.randomUUID?.() ??
      `schedule-${Date.now()}-${Math.random().toString(36).slice(2)}`,
    type: 'absolute',
    adjustment_type: allowFixedPrice ? 'fixed_price' : 'discount',
    price: allowFixedPrice ? 0 : undefined,
    discount_rate: allowFixedPrice ? undefined : 0,
    show_banner: true,
    start_at: Math.floor(now.getTime() / 1000),
    end_at: Math.floor(end.getTime() / 1000),
  }
}

type ScheduledPricingEditorProps = {
  schedules: PriceSchedule[]
  onChange: (schedules: PriceSchedule[]) => void
  allowFixedPrice: boolean
}

export function ScheduledPricingEditor(props: ScheduledPricingEditorProps) {
  const { t, i18n } = useTranslation()
  const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone
  const timezones = useMemo(
    () => [...new Set([browserTimezone, ...COMMON_TIMEZONES].filter(Boolean))],
    [browserTimezone]
  )
  const weekdayLabels = useMemo(() => {
    const formatter = new Intl.DateTimeFormat(
      toIntlLocale(i18n.resolvedLanguage || i18n.language),
      { weekday: 'short', timeZone: 'UTC' }
    )
    return WEEKDAYS.map((_, index) =>
      formatter.format(new Date(Date.UTC(2026, 7, 23 + index)))
    )
  }, [i18n.language, i18n.resolvedLanguage])

  const updateSchedule = (index: number, next: PriceSchedule) => {
    props.onChange(
      props.schedules.map((schedule, i) => (i === index ? next : schedule))
    )
  }

  return (
    <FieldGroup className='gap-4'>
      {props.schedules.map((schedule, index) => {
        const availableTimezones = schedule.timezone
          ? [...new Set([schedule.timezone, ...timezones])]
          : timezones
        const adjustmentType = schedule.adjustment_type ?? 'fixed_price'
        return (
          <div key={schedule.id} className='rounded-lg border p-4'>
            <div className='mb-4 flex items-center justify-between gap-3'>
              <div className='text-sm font-medium'>
                {t('Schedule')} {index + 1}
              </div>
              <Button
                type='button'
                variant='ghost'
                size='icon-sm'
                aria-label={t('Delete schedule')}
                title={t('Delete schedule')}
                onClick={() =>
                  props.onChange(props.schedules.filter((_, i) => i !== index))
                }
              >
                <HugeiconsIcon icon={Delete02Icon} strokeWidth={2} />
              </Button>
            </div>

            <FieldGroup className='gap-4'>
              <Field>
                <FieldLabel>{t('Activity pricing')}</FieldLabel>
                <ToggleGroup
                  value={[adjustmentType]}
                  onValueChange={(values) => {
                    const nextType = values.find(
                      (value) => value !== adjustmentType
                    )
                    if (nextType === 'fixed_price' && props.allowFixedPrice) {
                      updateSchedule(index, {
                        ...schedule,
                        adjustment_type: 'fixed_price',
                        price: schedule.price ?? 0,
                        discount_rate: undefined,
                      })
                    } else if (nextType === 'discount') {
                      updateSchedule(index, {
                        ...schedule,
                        adjustment_type: 'discount',
                        price: undefined,
                        discount_rate: schedule.discount_rate ?? 0.8,
                      })
                    }
                  }}
                  variant='outline'
                  className='w-full'
                >
                  <ToggleGroupItem
                    value='fixed_price'
                    disabled={!props.allowFixedPrice}
                    className='flex-1'
                  >
                    {t('Fixed activity price')}
                  </ToggleGroupItem>
                  <ToggleGroupItem value='discount' className='flex-1'>
                    {t('Percentage discount')}
                  </ToggleGroupItem>
                </ToggleGroup>
                {!props.allowFixedPrice && adjustmentType === 'fixed_price' && (
                  <FieldDescription className='text-destructive'>
                    {t(
                      'Fixed activity prices are only available for per-request billing.'
                    )}
                  </FieldDescription>
                )}
              </Field>

              <Field>
                <FieldLabel>{t('Schedule type')}</FieldLabel>
                <ToggleGroup
                  value={[schedule.type]}
                  onValueChange={(values) => {
                    const type = values.find((value) => value !== schedule.type)
                    if (type === 'absolute') {
                      const fresh = createSchedule(props.allowFixedPrice)
                      updateSchedule(index, {
                        ...fresh,
                        id: schedule.id,
                        adjustment_type: adjustmentType,
                        price:
                          adjustmentType === 'fixed_price'
                            ? (schedule.price ?? 0)
                            : undefined,
                        discount_rate:
                          adjustmentType === 'discount'
                            ? (schedule.discount_rate ?? 0)
                            : undefined,
                        show_banner: schedule.show_banner !== false,
                      })
                    } else if (type === 'weekly') {
                      updateSchedule(index, {
                        id: schedule.id,
                        type: 'weekly',
                        adjustment_type: adjustmentType,
                        price:
                          adjustmentType === 'fixed_price'
                            ? (schedule.price ?? 0)
                            : undefined,
                        discount_rate:
                          adjustmentType === 'discount'
                            ? (schedule.discount_rate ?? 0)
                            : undefined,
                        show_banner: schedule.show_banner !== false,
                        weekdays: [1, 2, 3, 4, 5],
                        start_minute: 0,
                        end_minute: 0,
                        timezone: browserTimezone || 'Asia/Shanghai',
                      })
                    }
                  }}
                  variant='outline'
                  className='w-full'
                >
                  <ToggleGroupItem value='absolute' className='flex-1'>
                    <HugeiconsIcon
                      icon={Calendar03Icon}
                      strokeWidth={2}
                      data-icon='inline-start'
                    />
                    {t('Fixed range')}
                  </ToggleGroupItem>
                  <ToggleGroupItem value='weekly' className='flex-1'>
                    <HugeiconsIcon
                      icon={Clock01Icon}
                      strokeWidth={2}
                      data-icon='inline-start'
                    />
                    {t('Weekly')}
                  </ToggleGroupItem>
                </ToggleGroup>
              </Field>

              {adjustmentType === 'fixed_price' ? (
                <Field>
                  <FieldLabel htmlFor={`${schedule.id}-activity-price`}>
                    {t('Activity price')}
                  </FieldLabel>
                  <InputGroup>
                    <InputGroupAddon>$</InputGroupAddon>
                    <InputGroupInput
                      id={`${schedule.id}-activity-price`}
                      type='number'
                      inputMode='decimal'
                      min='0'
                      step='any'
                      value={schedule.price ?? 0}
                      onChange={(event) =>
                        updateSchedule(index, {
                          ...schedule,
                          price: Number(event.target.value),
                        })
                      }
                    />
                    <InputGroupAddon align='inline-end'>
                      {t('per request')}
                    </InputGroupAddon>
                  </InputGroup>
                  <FieldDescription>
                    {t('Set the activity price to 0 for free usage.')}
                  </FieldDescription>
                </Field>
              ) : (
                <Field>
                  <FieldLabel htmlFor={`${schedule.id}-discount-rate`}>
                    {t('Discounted price percentage')}
                  </FieldLabel>
                  <InputGroup>
                    <InputGroupInput
                      id={`${schedule.id}-discount-rate`}
                      type='number'
                      inputMode='decimal'
                      min='0'
                      max='100'
                      step='any'
                      value={(schedule.discount_rate ?? 0) * 100}
                      onChange={(event) =>
                        updateSchedule(index, {
                          ...schedule,
                          discount_rate: Number(event.target.value) / 100,
                        })
                      }
                    />
                    <InputGroupAddon align='inline-end'>%</InputGroupAddon>
                  </InputGroup>
                  <FieldDescription>
                    {t('Enter 80 for 20% off. Set it to 0 for free usage.')}
                  </FieldDescription>
                </Field>
              )}

              <Field orientation='horizontal'>
                <FieldContent>
                  <FieldLabel htmlFor={`${schedule.id}-show-banner`}>
                    {t('Show in model activity banner')}
                  </FieldLabel>
                  <FieldDescription>
                    {t(
                      'Display this activity in the global model promotion banner while it is active.'
                    )}
                  </FieldDescription>
                </FieldContent>
                <Switch
                  id={`${schedule.id}-show-banner`}
                  checked={schedule.show_banner !== false}
                  onCheckedChange={(checked) =>
                    updateSchedule(index, {
                      ...schedule,
                      show_banner: checked,
                    })
                  }
                />
              </Field>

              {schedule.type === 'absolute' ? (
                <div className='grid gap-4 sm:grid-cols-2'>
                  <Field>
                    <FieldLabel htmlFor={`${schedule.id}-start`}>
                      {t('Start time')}
                    </FieldLabel>
                    <Input
                      id={`${schedule.id}-start`}
                      type='datetime-local'
                      value={formatDateTimeLocal(schedule.start_at)}
                      onChange={(event) =>
                        updateSchedule(index, {
                          ...schedule,
                          start_at: parseDateTimeLocal(event.target.value),
                        })
                      }
                    />
                  </Field>
                  <Field>
                    <FieldLabel htmlFor={`${schedule.id}-end`}>
                      {t('End time')}
                    </FieldLabel>
                    <Input
                      id={`${schedule.id}-end`}
                      type='datetime-local'
                      value={formatDateTimeLocal(schedule.end_at)}
                      onChange={(event) =>
                        updateSchedule(index, {
                          ...schedule,
                          end_at: parseDateTimeLocal(event.target.value),
                        })
                      }
                    />
                  </Field>
                </div>
              ) : (
                <FieldGroup className='gap-4'>
                  <Field>
                    <FieldLabel>{t('Weekdays')}</FieldLabel>
                    <ToggleGroup
                      value={(schedule.weekdays || []).map(String)}
                      onValueChange={(values) =>
                        updateSchedule(index, {
                          ...schedule,
                          weekdays: values.map(Number).sort((a, b) => a - b),
                        })
                      }
                      variant='outline'
                      size='sm'
                      className='grid w-full grid-cols-4 sm:grid-cols-7'
                    >
                      {WEEKDAYS.map((weekday, weekdayIndex) => (
                        <ToggleGroupItem
                          key={weekday}
                          value={String(weekdayIndex)}
                          className='w-full'
                          aria-label={t(weekday)}
                          title={t(weekday)}
                        >
                          {weekdayLabels[weekdayIndex]}
                        </ToggleGroupItem>
                      ))}
                    </ToggleGroup>
                  </Field>

                  <div className='grid gap-4 sm:grid-cols-2'>
                    <Field>
                      <FieldLabel htmlFor={`${schedule.id}-weekly-start`}>
                        {t('Start time')}
                      </FieldLabel>
                      <Input
                        id={`${schedule.id}-weekly-start`}
                        type='time'
                        value={formatTime(schedule.start_minute)}
                        onChange={(event) =>
                          updateSchedule(index, {
                            ...schedule,
                            start_minute: parseTime(event.target.value),
                          })
                        }
                      />
                    </Field>
                    <Field>
                      <FieldLabel htmlFor={`${schedule.id}-weekly-end`}>
                        {t('End time')}
                      </FieldLabel>
                      <Input
                        id={`${schedule.id}-weekly-end`}
                        type='time'
                        value={formatTime(schedule.end_minute)}
                        onChange={(event) =>
                          updateSchedule(index, {
                            ...schedule,
                            end_minute: parseTime(event.target.value),
                          })
                        }
                      />
                    </Field>
                  </div>
                  <FieldDescription>
                    {t(
                      'Matching start and end times mean all day. Overnight ranges continue into the next day.'
                    )}
                  </FieldDescription>

                  <Field>
                    <FieldLabel>{t('Timezone')}</FieldLabel>
                    <Select
                      value={schedule.timezone || null}
                      onValueChange={(value) => {
                        if (typeof value === 'string') {
                          updateSchedule(index, {
                            ...schedule,
                            timezone: value,
                          })
                        }
                      }}
                    >
                      <SelectTrigger className='w-full'>
                        <SelectValue placeholder={t('Select a timezone')} />
                      </SelectTrigger>
                      <SelectContent alignItemWithTrigger={false}>
                        <SelectGroup>
                          {availableTimezones.map((timezone) => (
                            <SelectItem key={timezone} value={timezone}>
                              {timezone}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  </Field>
                </FieldGroup>
              )}
            </FieldGroup>
          </div>
        )
      })}

      <Button
        type='button'
        variant='outline'
        onClick={() =>
          props.onChange([
            ...props.schedules,
            createSchedule(props.allowFixedPrice),
          ])
        }
      >
        <HugeiconsIcon
          icon={Add01Icon}
          strokeWidth={2}
          data-icon='inline-start'
        />
        {t('Add schedule')}
      </Button>
    </FieldGroup>
  )
}
