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

import { redemptionTopupFrameProps } from '../layout.ts'

describe('redemption top-up layout', () => {
  test('fills the content area with the sandboxed redemption shop', () => {
    const frameClasses = redemptionTopupFrameProps.className.split(' ')
    const sandboxPermissions = redemptionTopupFrameProps.sandbox.split(' ')

    assert.equal(
      redemptionTopupFrameProps.src,
      'https://catfk.com/shop/40RJHCKS'
    )
    assert.ok(frameClasses.includes('h-full'))
    assert.ok(frameClasses.includes('w-full'))
    assert.ok(frameClasses.includes('flex-1'))
    assert.ok(sandboxPermissions.includes('allow-forms'))
    assert.ok(sandboxPermissions.includes('allow-scripts'))
    assert.ok(sandboxPermissions.includes('allow-same-origin'))
    assert.equal(sandboxPermissions.includes('allow-top-navigation'), false)
    assert.equal(redemptionTopupFrameProps.allow, 'payment')
    assert.equal(
      redemptionTopupFrameProps.referrerPolicy,
      'strict-origin-when-cross-origin'
    )
  })
})
