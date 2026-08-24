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
  applyResolutionSelections,
  deleteResolutionField,
  formatSyncValue,
  getEffectiveResolutionSelections,
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

  test('token discount schedules stay independent from token ratios', () => {
    const sourceName = 'member-node'
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
    const differences: DifferencesMap = {
      'token-model': {
        model_ratio: {
          current: 1,
          upstreams: { [sourceName]: 2 },
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
        differences['token-model'] ?? {},
        sourceName
      ),
      ['model_ratio', 'price_schedules']
    )
    assert.deepEqual(
      applyResolutionSelections({}, differences, [
        {
          model: 'token-model',
          ratioType: 'model_ratio',
          value: 2,
          sourceName,
        },
        {
          model: 'token-model',
          ratioType: 'price_schedules',
          value: schedules,
          sourceName,
        },
      ]),
      {
        'token-model': {
          model_ratio: 2,
          price_schedules: schedules,
        },
      }
    )
    assert.deepEqual(
      getEffectiveResolutionSelections(differences, [
        {
          model: 'token-model',
          ratioType: 'price_schedules',
          value: schedules,
          sourceName,
        },
        {
          model: 'token-model',
          ratioType: 'model_ratio',
          value: 2,
          sourceName,
        },
      ]).map(({ ratioType }) => ratioType),
      ['price_schedules', 'model_ratio']
    )
    assert.deepEqual(
      applyResolutionSelection(
        { 'token-model': { model_ratio: 1 } },
        differences,
        {
          model: 'token-model',
          ratioType: 'price_schedules',
          value: schedules,
          sourceName,
        }
      ),
      {
        'token-model': {
          model_ratio: 1,
          price_schedules: schedules,
        },
      }
    )
  })

  test('selecting a token ratio removes a fixed-price activity from another source', () => {
    const fixedSource = 'fixed-price-node'
    const ratioSource = 'token-ratio-node'
    const schedules = [
      {
        id: 'legacy-fixed-hours',
        type: 'absolute',
        price: 0.1,
        start_at: 100,
        end_at: 200,
      },
    ]
    const differences: DifferencesMap = {
      'mixed-model': {
        model_ratio: {
          current: null,
          upstreams: { [ratioSource]: 2 },
          confidence: { [ratioSource]: true },
        },
        billing_mode: {
          current: null,
          upstreams: { [fixedSource]: 'scheduled_price' },
          confidence: { [fixedSource]: true },
        },
        model_price: {
          current: null,
          upstreams: { [fixedSource]: 0.5 },
          confidence: { [fixedSource]: true },
        },
        price_schedules: {
          current: null,
          upstreams: { [fixedSource]: schedules },
          confidence: { [fixedSource]: true },
        },
      },
    }
    const selections = [
      {
        model: 'mixed-model',
        ratioType: 'price_schedules' as const,
        value: schedules,
        sourceName: fixedSource,
      },
      {
        model: 'mixed-model',
        ratioType: 'model_ratio' as const,
        value: 2,
        sourceName: ratioSource,
      },
    ]

    assert.deepEqual(applyResolutionSelections({}, differences, selections), {
      'mixed-model': { model_ratio: 2 },
    })
    assert.deepEqual(
      getEffectiveResolutionSelections(differences, selections).map(
        ({ ratioType }) => ratioType
      ),
      ['model_ratio']
    )
  })
})
