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
import {
  applyResolutionSelection,
  getBillingCategory,
} from '../upstream-ratio-sync-helpers'
import {
  isVideoResolutionPricesMap,
  parseVideoResolutionPriceDraft,
} from '../video-resolution-prices'

const base = {
  modelPrice: '{"video":0.5}',
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

describe('video quality prices', () => {
  test('accepts all five tiers, preserves free prices, and omits blank fallbacks', () => {
    assert.deepEqual(
      parseVideoResolutionPriceDraft({
        '480p': '0',
        '720p': '0.1',
        '768p': '.2',
        '1080p': '0.3',
        '2k': '0.4',
      }),
      { '480p': 0, '720p': 0.1, '768p': 0.2, '1080p': 0.3, '2k': 0.4 }
    )
    assert.deepEqual(parseVideoResolutionPriceDraft({ '720p': '' }), {})
    for (const value of ['-1', '.', 'Infinity', 'NaN', '1e400', '0x10']) {
      assert.equal(parseVideoResolutionPriceDraft({ '720p': value }), null)
    }
    assert.equal(isVideoResolutionPricesMap({ video: { '720p': null } }), false)
    assert.equal(isVideoResolutionPricesMap({ video: { '4k': 1 } }), false)
  })
  test('reloads stored quality prices and marks a quality-only edit as changed', () => {
    const [snapshot] = buildModelSnapshots({
      ...base,
      videoResolutionPrices: '{"video":{"720p":0,"768p":0.2,"2k":0.4}}',
    })
    assert.deepEqual(snapshot.videoResolutionPrices, {
      '720p': 0,
      '768p': 0.2,
      '2k': 0.4,
    })
    assert.notEqual(
      getSnapshotSignature(snapshot),
      getSnapshotSignature({ ...snapshot, videoResolutionPrices: {} })
    )
  })
  test('syncs quality prices without discarding an existing fixed price', () => {
    const prices = { '720p': 0.1, '768p': 0.2 }
    const result = applyResolutionSelection(
      { video: { model_price: 0.5 } },
      {
        video: {
          video_resolution_prices: {
            current: null,
            upstreams: { upstream: prices },
            confidence: { upstream: true },
          },
        },
      },
      {
        model: 'video',
        ratioType: 'video_resolution_prices',
        value: prices,
        sourceName: 'upstream',
      }
    )
    assert.equal(
      getBillingCategory('video_resolution_prices', prices),
      'activity'
    )
    assert.deepEqual(result, {
      video: { model_price: 0.5, video_resolution_prices: prices },
    })
  })
})
