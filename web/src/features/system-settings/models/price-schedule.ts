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
import type { PriceSchedule } from './model-pricing-core'

export function validatePriceSchedules(
  schedules: PriceSchedule[],
  t: (key: string) => string,
  allowFixedPrice = true
) {
  if (schedules.length === 0) return t('Add at least one price schedule.')
  if (schedules.length > 64) return t('A model can have at most 64 schedules.')

  for (const schedule of schedules) {
    const adjustmentType = schedule.adjustment_type ?? 'fixed_price'
    if (adjustmentType === 'fixed_price') {
      if (!allowFixedPrice) {
        return t(
          'Fixed activity prices are only available for per-request billing.'
        )
      }
      if (!Number.isFinite(schedule.price) || (schedule.price ?? -1) < 0) {
        return t('Activity price must be zero or greater.')
      }
    } else if (
      !Number.isFinite(schedule.discount_rate) ||
      (schedule.discount_rate ?? -1) < 0 ||
      (schedule.discount_rate ?? 2) > 1
    ) {
      return t('Discount rate must be between 0% and 100%.')
    }
    if (schedule.type === 'absolute') {
      if (
        !schedule.start_at ||
        !schedule.end_at ||
        schedule.end_at <= schedule.start_at
      ) {
        return t('The end time must be later than the start time.')
      }
      continue
    }
    if (!schedule.weekdays?.length) {
      return t('Select at least one weekday.')
    }
    if (!schedule.timezone) return t('Select a timezone')
  }
  return null
}
