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

import type { DifferencesMap } from '../../types'
import {
  applyResolutionSelection,
  deleteResolutionField,
  formatSyncValue,
  getVisibleRatioTypesForSource,
} from '../upstream-ratio-sync-helpers'

describe('upstream scheduled pricing sync', () => {
  test('selecting a scheduled price carries mode, base price, and schedules together', () => {
    const sourceName = 'member-node'
    const schedules = [
      {
        id: 'free-hours',
        type: 'weekly',
        price: 0,
        weekdays: [1, 5],
        start_minute: 1200,
        end_minute: 1320,
        timezone: 'Asia/Shanghai',
      },
    ]
    const differences: DifferencesMap = {
      'video-model': {
        billing_mode: {
          current: null,
          upstreams: { [sourceName]: 'scheduled_price' },
          confidence: { [sourceName]: true },
        },
        model_price: {
          current: 0.5,
          upstreams: { [sourceName]: 0.25 },
          confidence: { [sourceName]: true },
        },
        price_schedules: {
          current: null,
          upstreams: { [sourceName]: schedules },
          confidence: { [sourceName]: true },
        },
      },
    }

    assert.deepEqual(
      getVisibleRatioTypesForSource(
        differences['video-model'] ?? {},
        sourceName
      ),
      ['price_schedules']
    )

    const resolutions = applyResolutionSelection({}, differences, {
      model: 'video-model',
      ratioType: 'model_price',
      value: 0.25,
      sourceName,
    })

    assert.deepEqual(resolutions['video-model'], {
      price_schedules: schedules,
      billing_mode: 'scheduled_price',
      model_price: 0.25,
    })
    assert.equal(formatSyncValue(schedules), JSON.stringify(schedules))
    assert.deepEqual(
      deleteResolutionField(resolutions, 'video-model', 'price_schedules'),
      {}
    )
  })
})
