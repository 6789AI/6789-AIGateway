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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { validatePriceSchedules } from '../price-schedule'

const t = (key: string) => key

describe('time-based model price validation', () => {
  test('accepts free fixed-range and overnight weekly schedules', () => {
    assert.equal(
      validatePriceSchedules(
        [
          {
            id: 'fixed',
            type: 'absolute',
            price: 0,
            start_at: 100,
            end_at: 200,
          },
          {
            id: 'weekly',
            type: 'weekly',
            price: 0,
            weekdays: [1, 5],
            start_minute: 22 * 60,
            end_minute: 2 * 60,
            timezone: 'Asia/Shanghai',
          },
        ],
        t
      ),
      null
    )
  })

  test('rejects an empty schedule list', () => {
    assert.equal(
      validatePriceSchedules([], t),
      'Add at least one price schedule.'
    )
  })

  test('rejects a fixed range whose end is not later than its start', () => {
    assert.equal(
      validatePriceSchedules(
        [
          {
            id: 'fixed',
            type: 'absolute',
            price: 1,
            start_at: 200,
            end_at: 100,
          },
        ],
        t
      ),
      'The end time must be later than the start time.'
    )
  })

  test('rejects a weekly schedule without selected weekdays', () => {
    assert.equal(
      validatePriceSchedules(
        [
          {
            id: 'weekly',
            type: 'weekly',
            price: 1,
            weekdays: [],
            start_minute: 0,
            end_minute: 0,
            timezone: 'UTC',
          },
        ],
        t
      ),
      'Select at least one weekday.'
    )
  })
})
