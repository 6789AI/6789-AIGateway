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

import type { Channel } from '../../types'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  getDefaultVideoProtocol,
  supportsVideoProtocolConfiguration,
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
  type ChannelFormValues,
} from '../channel-form'

function validForm(values: Partial<ChannelFormValues>): ChannelFormValues {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'video channel',
    key: 'secret',
    models: 'seedance2.0',
    group: ['default'],
    ...values,
  }
}

function channelFixture(settings: Record<string, unknown>): Channel {
  return {
    id: 89,
    type: 55,
    name: 'Vinted video',
    status: 1,
    key: '',
    base_url: 'https://vinted.cam',
    models: 'seedance2.0',
    group: 'default',
    priority: 0,
    weight: 0,
    auto_ban: 1,
    setting: '{}',
    settings: JSON.stringify(settings),
    other: '',
    channel_info: { is_multi_key: false, multi_key_mode: 'random' },
  } as Channel
}

describe('video protocol settings', () => {
  test('supports only OpenAI and Sora channel types', () => {
    assert.equal(supportsVideoProtocolConfiguration(1), true)
    assert.equal(supportsVideoProtocolConfiguration(55), true)
    assert.equal(supportsVideoProtocolConfiguration(14), false)
  })

  test('detects legacy Vinted channels by hostname', () => {
    assert.equal(getDefaultVideoProtocol(55, 'https://vinted.cam'), 'vinted')
    assert.equal(
      getDefaultVideoProtocol(1, 'https://api.vinted.cam/base'),
      'vinted'
    )
    assert.equal(getDefaultVideoProtocol(55, 'https://example.com'), 'openai')
  })

  test('loads and serializes the explicit Vinted protocol', () => {
    const defaults = transformChannelToFormDefaults(
      channelFixture({ video_protocol: 'vinted' })
    )
    assert.equal(defaults.video_protocol, 'vinted')

    const payload = transformFormDataToCreatePayload(
      validForm({ type: 55, video_protocol: 'vinted' })
    )
    const settings = JSON.parse(String(payload.channel.settings))
    assert.equal(settings.video_protocol, 'vinted')
  })

  test('keeps an explicit OpenAI protocol on a legacy Vinted hostname', () => {
    const defaults = transformChannelToFormDefaults(
      channelFixture({ video_protocol: 'openai' })
    )
    assert.equal(defaults.video_protocol, 'openai')

    const payload = transformFormDataToCreatePayload(
      validForm({
        type: 55,
        base_url: 'https://vinted.cam',
        video_protocol: 'openai',
      })
    )
    const settings = JSON.parse(String(payload.channel.settings))
    assert.equal(settings.video_protocol, 'openai')
  })

  test('removes the protocol from unsupported channel types', () => {
    const payload = transformFormDataToCreatePayload(
      validForm({ type: 14, video_protocol: 'vinted' })
    )
    const settings = JSON.parse(String(payload.channel.settings))
    assert.equal('video_protocol' in settings, false)
  })
})
