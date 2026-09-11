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

import {
  applyResolutionSelection,
  applyResolutionSelections,
  getBillingCategory,
  getPreferredSyncField,
  getVisibleRatioTypesForSource,
  type RatioDifferenceEntry,
} from '../upstream-ratio-sync-helpers'

describe('task duration multiplier upstream sync', () => {
  test('treats the setting as an adjunct to fixed pricing', () => {
    assert.equal(
      getBillingCategory('task_duration_multiplier', false),
      'activity'
    )
  })

  test('selecting the setting preserves an existing fixed-price resolution', () => {
    const durationDifference: RatioDifferenceEntry = {
      current: true,
      upstreams: { upstream: false },
      confidence: { upstream: true },
    }
    const resolutions = applyResolutionSelection(
      { 'video-model': { model_price: 0.2 } },
      {
        'video-model': {
          task_duration_multiplier: durationDifference,
        },
      },
      {
        model: 'video-model',
        ratioType: 'task_duration_multiplier',
        value: false,
        sourceName: 'upstream',
      }
    )

    assert.deepEqual(resolutions, {
      'video-model': {
        model_price: 0.2,
        task_duration_multiplier: false,
      },
    })
  })

  test('keeps the setting independently selectable for scheduled fixed pricing', () => {
    const sourceName = 'upstream'
    const schedules = [
      {
        id: 'discount-hours',
        type: 'absolute',
        adjustment_type: 'discount',
        discount_rate: 0.8,
        start_at: 100,
        end_at: 200,
      },
    ]
    const differences = {
      'video-model': {
        billing_mode: {
          current: null,
          upstreams: { [sourceName]: 'scheduled_price' as const },
          confidence: { [sourceName]: true },
        },
        model_price: {
          current: 0.5,
          upstreams: { [sourceName]: 0.2 },
          confidence: { [sourceName]: true },
        },
        price_schedules: {
          current: null,
          upstreams: { [sourceName]: schedules },
          confidence: { [sourceName]: true },
        },
        task_duration_multiplier: {
          current: true,
          upstreams: { [sourceName]: false },
          confidence: { [sourceName]: true },
        },
      },
    }

    assert.equal(
      getPreferredSyncField(
        differences['video-model'],
        'task_duration_multiplier',
        sourceName
      ),
      'task_duration_multiplier'
    )
    assert.deepEqual(
      getVisibleRatioTypesForSource(differences['video-model'], sourceName),
      ['price_schedules', 'task_duration_multiplier']
    )
    assert.deepEqual(
      applyResolutionSelections({}, differences, [
        {
          model: 'video-model',
          ratioType: 'price_schedules',
          value: schedules,
          sourceName,
        },
        {
          model: 'video-model',
          ratioType: 'task_duration_multiplier',
          value: false,
          sourceName,
        },
      ]),
      {
        'video-model': {
          billing_mode: 'scheduled_price',
          model_price: 0.2,
          price_schedules: schedules,
          task_duration_multiplier: false,
        },
      }
    )
  })
})
