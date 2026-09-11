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

const baseInput = {
  modelPrice: '{"video-model":0.2}',
  modelRatio: '{}',
  cacheRatio: '{}',
  createCacheRatio: '{}',
  completionRatio: '{}',
  imageRatio: '{}',
  audioRatio: '{}',
  audioCompletionRatio: '{}',
  billingMode: '{}',
  billingExpr: '{}',
  priceSchedules: '{}',
}

describe('task duration multiplier pricing snapshots', () => {
  test('defaults existing fixed-price models to multiplying by duration', () => {
    const [snapshot] = buildModelSnapshots(baseInput)

    assert.equal(snapshot.multiplyByDuration, true)
  })

  test('loads an explicit disabled duration multiplier', () => {
    const [snapshot] = buildModelSnapshots({
      ...baseInput,
      taskDurationMultiplier: '{"video-model":false}',
    })

    assert.equal(snapshot.multiplyByDuration, false)
  })

  test('includes the duration multiplier in draft change detection', () => {
    const [snapshot] = buildModelSnapshots(baseInput)

    assert.notEqual(
      getSnapshotSignature(snapshot),
      getSnapshotSignature({ ...snapshot, multiplyByDuration: false })
    )
  })
})
