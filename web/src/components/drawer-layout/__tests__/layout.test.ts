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

import { cn } from '../../../lib/utils.ts'
import { sideDrawerContentClassName } from '../../drawer-layout-classes.ts'

describe('side drawer content layout', () => {
  test('does not impose viewport height over the sheet banner offset', () => {
    const classes = cn(
      'fixed top-[var(--global-banner-height,0px)] bottom-0 h-[calc(100dvh-var(--global-banner-height,0px))]',
      sideDrawerContentClassName()
    )

    assert.match(
      classes,
      /h-\[calc\(100dvh-var\(--global-banner-height,0px\)\)\]/
    )
    assert.doesNotMatch(classes, /h-dvh/)
  })
})
