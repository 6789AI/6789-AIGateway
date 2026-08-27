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
  channelFormSchema,
  getDefaultAsyncImageProvider,
  supportsAsyncImageConfiguration,
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
  type ChannelFormValues,
} from '../channel-form'

function validForm(
  values: Partial<ChannelFormValues> = {}
): ChannelFormValues {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'image channel',
    key: 'secret',
    models: 'image-model',
    group: ['default'],
    ...values,
  }
}

function channelFixture(settings: Record<string, unknown>): Channel {
  return {
    id: 1,
    type: 8,
    name: 'custom image',
    status: 1,
    key: '',
    base_url: 'https://grsaiapi.com',
    models: 'image-model',
    group: 'default',
    priority: 0,
    weight: 0,
    auto_ban: 1,
    setting: '{}',
    settings: JSON.stringify(settings),
    other: '',
    channel_info: {
      is_multi_key: false,
      multi_key_mode: 'random',
    },
  } as Channel
}

describe('async image channel settings', () => {
  test('keeps asynchronous image generation disabled by default', () => {
    assert.equal(CHANNEL_FORM_DEFAULT_VALUES.async_image_enabled, false)
    assert.equal(CHANNEL_FORM_DEFAULT_VALUES.async_image_provider, 'new_api')
  })

  test('smart-selects the initial protocol from channel type and base URL', () => {
    assert.equal(getDefaultAsyncImageProvider(17, 'https://example.com'), 'ali')
    assert.equal(
      getDefaultAsyncImageProvider(8, 'https://grsaiapi.com'),
      'grsai'
    )
    assert.equal(
      getDefaultAsyncImageProvider(60, 'https://new-api.example.com'),
      'new_api'
    )
  })

  test('exposes configuration only for documented image channel types', () => {
    assert.equal(supportsAsyncImageConfiguration(1), true)
    assert.equal(supportsAsyncImageConfiguration(8), true)
    assert.equal(supportsAsyncImageConfiguration(58), true)
    assert.equal(supportsAsyncImageConfiguration(60), true)
    assert.equal(supportsAsyncImageConfiguration(14), false)
  })

  test('requires a protocol and supported channel type when enabled', () => {
    const missingProtocol = channelFormSchema.safeParse(
      validForm({ async_image_enabled: true, async_image_provider: undefined })
    )
    assert.equal(missingProtocol.success, false)

    const unsupportedType = channelFormSchema.safeParse(
      validForm({
        type: 14,
        async_image_enabled: true,
        async_image_provider: 'new_api',
      })
    )
    assert.equal(unsupportedType.success, false)
  })

  test('keeps a legacy provider selected but does not enable it', () => {
    const defaults = transformChannelToFormDefaults(
      channelFixture({ async_image_provider: 'grsai' })
    )
    assert.equal(defaults.async_image_enabled, false)
    assert.equal(defaults.async_image_provider, 'grsai')
    assert.equal(defaults.async_image_provider_explicit, true)
  })

  test('serializes enabled settings and removes them for unsupported types', () => {
    const enabledPayload = transformFormDataToCreatePayload(
      validForm({
        type: 60,
        base_url: 'https://new-api.example.com',
        async_image_enabled: true,
        async_image_provider: 'new_api',
      })
    )
    const enabledSettings = JSON.parse(
      String(enabledPayload.channel.settings)
    )
    assert.equal(enabledSettings.async_image_enabled, true)
    assert.equal(enabledSettings.async_image_provider, 'new_api')

    const unsupportedPayload = transformFormDataToCreatePayload(
      validForm({
        type: 14,
        async_image_enabled: false,
        async_image_provider: 'grsai',
      })
    )
    const unsupportedSettings = JSON.parse(
      String(unsupportedPayload.channel.settings)
    )
    assert.equal('async_image_enabled' in unsupportedSettings, false)
    assert.equal('async_image_provider' in unsupportedSettings, false)
  })
})
