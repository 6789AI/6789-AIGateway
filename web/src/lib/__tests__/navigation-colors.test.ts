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

import { parseSidebarNavColors } from '../navigation-colors.ts'

describe('sidebar navigation colors', () => {
  test('keeps configured link and chat colors for sidebar rendering', () => {
    const colors = parseSidebarNavColors(
      JSON.stringify({ '/wallet': '#E07814', 'chat-presets': '#1275EE' })
    )

    assert.deepEqual(colors, {
      '/wallet': '#E07814',
      'chat-presets': '#1275EE',
    })
  })

  test('ignores malformed colors and unsupported keys from saved status', () => {
    const colors = parseSidebarNavColors(
      JSON.stringify({
        '/wallet': 'var(--color)',
        '/keys': '#123456',
        'invalid-key': '#FF0000',
      })
    )

    assert.deepEqual(colors, { '/keys': '#123456' })
  })
})
