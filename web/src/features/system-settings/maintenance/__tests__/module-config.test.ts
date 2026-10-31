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

import { parseHeaderNavModules, parseSidebarModulesAdmin } from '../config.ts'

describe('navigation module configuration', () => {
  test('shows Comfy Canvas by default for legacy header configuration', () => {
    const config = parseHeaderNavModules('{"home":true}')

    assert.equal(config.comfyCanvas, true)
  })

  test('preserves a disabled Comfy Canvas setting', () => {
    const config = parseHeaderNavModules('{"comfyCanvas":false}')

    assert.equal(config.comfyCanvas, false)
  })

  test('loads ordered custom links while preserving legacy built-in entries', () => {
    const config = parseHeaderNavModules(
      JSON.stringify({
        home: false,
        links: [
          {
            key: 'custom-help',
            label: 'Help',
            href: 'https://example.com/help',
          },
          { key: 'docs', label: 'Guide', href: '/guide' },
        ],
      })
    )

    assert.equal(config.home, false)
    assert.deepEqual(
      config.links.slice(0, 2).map((link) => link.key),
      ['custom-help', 'docs']
    )
    assert.equal(config.links.find((link) => link.key === 'home')?.href, '')
  })

  test('drops unsafe and duplicate custom URLs from saved settings', () => {
    const config = parseHeaderNavModules(
      JSON.stringify({
        links: [
          { key: 'custom-bad', label: 'Bad', href: 'javascript:alert(1)' },
          { key: 'custom-good', label: 'Good', href: '/good' },
          { key: 'custom-good', label: 'Duplicate', href: '/duplicate' },
          {
            key: 'docs',
            label: 'Docs',
            href: 'https://user:pass@example.com/',
          },
        ],
      })
    )

    assert.equal(
      config.links.some((link) => link.key === 'custom-bad'),
      false
    )
    assert.equal(
      config.links.filter((link) => link.key === 'custom-good').length,
      1
    )
    assert.equal(config.links.find((link) => link.key === 'docs')?.href, '')
  })

  test('shows redemption top-up by default for legacy sidebar configuration', () => {
    const config = parseSidebarModulesAdmin(
      '{"personal":{"enabled":true,"topup":true,"personal":true}}'
    )

    assert.equal(config.personal.redemptionTopup, true)
  })

  test('preserves a disabled redemption top-up setting', () => {
    const config = parseSidebarModulesAdmin(
      '{"personal":{"enabled":true,"redemptionTopup":false}}'
    )

    assert.equal(config.personal.redemptionTopup, false)
  })
})
