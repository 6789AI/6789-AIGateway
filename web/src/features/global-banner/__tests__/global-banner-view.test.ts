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

import { getGlobalBannerView } from '../use-global-banner'

describe('global banner view', () => {
  test('shows a configured marketing banner without a free model promotion', () => {
    const view = getGlobalBannerView(
      {
        active: false,
        marketing_banner: {
          enabled: true,
          content: 'Weekend bonus',
          background_color: '#112233',
          text_color: '#FFFFFF',
          icon: 'gift',
          countdown_enabled: false,
          countdown_end_at: 0,
          link_url: '/wallet',
        },
        server_time: 1_000,
        models: [],
      },
      10_000,
      10_000
    )

    assert.equal(view.visible, true)
    assert.equal(view.marketingBanner.content, 'Weekend bonus')
    assert.deepEqual(view.models, [])
  })

  test('filters expired promotions using elapsed server time', () => {
    const view = getGlobalBannerView(
      {
        active: true,
        marketing_banner: {
          enabled: false,
          content: '',
          background_color: '#A3E635',
          text_color: '#1A2E05',
          icon: 'megaphone',
          countdown_enabled: false,
          countdown_end_at: 0,
          link_url: '',
        },
        server_time: 1_000,
        models: [
          { model_name: 'expired', ends_at: 1_001 },
          { model_name: 'active', ends_at: 1_010 },
        ],
      },
      10_000,
      12_000
    )

    assert.equal(view.visible, true)
    assert.deepEqual(view.models, [{ model_name: 'active', ends_at: 1_010 }])
    assert.equal(view.promotionRemainingSeconds, 8)
  })

  test('hides an expired marketing countdown without hiding free models', () => {
    const view = getGlobalBannerView(
      {
        active: true,
        marketing_banner: {
          enabled: true,
          content: 'Expired campaign',
          background_color: '#112233',
          text_color: '#FFFFFF',
          icon: 'bell',
          countdown_enabled: true,
          countdown_end_at: 1_005,
          link_url: 'https://example.com/promotion',
        },
        server_time: 1_000,
        models: [{ model_name: 'still-free', ends_at: 1_020 }],
      },
      10_000,
      16_000
    )

    assert.equal(view.visible, true)
    assert.equal(view.marketingBanner.enabled, false)
    assert.equal(view.marketingRemainingSeconds, null)
    assert.deepEqual(view.models, [
      { model_name: 'still-free', ends_at: 1_020 },
    ])
  })
})
