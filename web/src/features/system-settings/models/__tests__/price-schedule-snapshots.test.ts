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
  buildModelSnapshots,
  getSnapshotSignature,
} from '../model-pricing-snapshots'

const emptyRatios = {
  modelRatio: '{}',
  cacheRatio: '{}',
  createCacheRatio: '{}',
  completionRatio: '{}',
  imageRatio: '{}',
  audioRatio: '{}',
  audioCompletionRatio: '{}',
  billingExpr: '{}',
}

describe('time-based model price snapshots', () => {
  test('loads the base price and all schedules for a scheduled model', () => {
    const [snapshot] = buildModelSnapshots({
      ...emptyRatios,
      modelPrice: '{"video-model":0.25}',
      billingMode: '{"video-model":"scheduled_price"}',
      priceSchedules:
        '{"video-model":[{"id":"free-weekend","type":"weekly","price":0,"weekdays":[0,6],"start_minute":0,"end_minute":0,"timezone":"Asia/Shanghai"}]}',
    })

    assert.equal(snapshot?.billingMode, 'scheduled_price')
    assert.equal(snapshot?.price, '0.25')
    assert.equal(snapshot?.priceSchedules?.length, 1)
    assert.equal(snapshot?.priceSchedules?.[0]?.price, 0)
  })

  test('schedule changes are included in the draft signature', () => {
    const [snapshot] = buildModelSnapshots({
      ...emptyRatios,
      modelPrice: '{"video-model":0.25}',
      billingMode: '{"video-model":"scheduled_price"}',
      priceSchedules:
        '{"video-model":[{"id":"activity","type":"absolute","price":0.1,"start_at":100,"end_at":200}]}',
    })
    assert.ok(snapshot)

    const changed = {
      ...snapshot,
      priceSchedules: snapshot.priceSchedules?.map((schedule) => ({
        ...schedule,
        price: 0,
      })),
    }

    assert.notEqual(
      getSnapshotSignature(snapshot),
      getSnapshotSignature(changed)
    )
  })
})
