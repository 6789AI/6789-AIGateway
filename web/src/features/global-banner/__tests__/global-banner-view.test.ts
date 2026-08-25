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
  test('does not show the free-model style without an active promotion', () => {
    const view = getGlobalBannerView(
      {
        active: false,
        global_banner: {
          enabled: false,
          content: '',
          background_color: '#0EA5E9',
          text_color: '#082F49',
          icon: 'gift',
          countdown_enabled: false,
          countdown_end_at: 0,
          link_url: '/pricing',
          button_text: '',
          button_color: '#FFFFFF',
        },
        free_model_banner: {
          background_color: '#112233',
          text_color: '#FFFFFF',
          icon: 'gift',
          link_url: '/wallet',
          button_text: 'Open wallet',
          button_color: '#123456',
        },
        server_time: 1_000,
        models: [],
      },
      10_000,
      10_000
    )

    assert.equal(view.visible, false)
    assert.equal(view.freeModelBanner.link_url, '/wallet')
    assert.equal(view.freeModelBanner.button_text, 'Open wallet')
    assert.deepEqual(view.models, [])
  })

  test('treats null promotion data as an empty list', () => {
    const view = getGlobalBannerView(
      {
        active: false,
        free_model_banner: {
          background_color: '#A3E635',
          text_color: '#1A2E05',
          icon: 'megaphone',
          link_url: '',
          button_text: '',
          button_color: '#FFFFFF',
        },
        server_time: 1_000,
        models: null,
      },
      10_000,
      10_000
    )

    assert.equal(view.visible, false)
    assert.deepEqual(view.models, [])
  })

  test('filters expired promotions using elapsed server time', () => {
    const view = getGlobalBannerView(
      {
        active: true,
        global_banner: {
          enabled: false,
          content: '',
          background_color: '#0EA5E9',
          text_color: '#082F49',
          icon: 'gift',
          countdown_enabled: false,
          countdown_end_at: 0,
          link_url: '/pricing',
          button_text: '',
          button_color: '#FFFFFF',
        },
        free_model_banner: {
          background_color: '#A3E635',
          text_color: '#1A2E05',
          icon: 'megaphone',
          link_url: '',
          button_text: '',
          button_color: '#FFFFFF',
        },
        server_time: 1_000,
        models: [
          { model_name: 'expired', promotion_type: 'free', ends_at: 1_001 },
          {
            model_name: 'active',
            promotion_type: 'discount',
            discount_rate: 0.8,
            ends_at: 1_010,
          },
        ],
      },
      10_000,
      12_000
    )

    assert.equal(view.visible, true)
    assert.equal(view.globalBanner.background_color, '#0EA5E9')
    assert.deepEqual(view.models, [
      {
        model_name: 'active',
        promotion_type: 'discount',
        discount_rate: 0.8,
        ends_at: 1_010,
      },
    ])
    assert.equal(view.promotionRemainingSeconds, 8)
  })
})
