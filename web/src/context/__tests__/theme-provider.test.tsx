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
import { after, beforeEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window({ url: 'http://localhost/' })
const domGlobals = [
  'window',
  'document',
  'HTMLElement',
  'Node',
  'Element',
  'Event',
  'MutationObserver',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { ThemeProvider, useTheme } = await import('../theme-provider')

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

function ThemeState() {
  const theme = useTheme()
  return (
    <output
      data-theme={theme.theme}
      data-resolved-theme={theme.resolvedTheme}
      data-default-theme={theme.defaultTheme}
    />
  )
}

describe('theme provider default', () => {
  beforeEach(() => {
    document.cookie = 'vite-ui-theme=; path=/; max-age=0'
    document.documentElement.classList.remove('light', 'dark')
  })

  after(() => {
    domWindow.close()
  })

  test('uses light mode when no preference has been saved', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <ThemeProvider>
          <ThemeState />
        </ThemeProvider>
      )
    })

    const output = container.querySelector('output')
    assert.ok(output)
    assert.equal(output.dataset.theme, 'light')
    assert.equal(output.dataset.resolvedTheme, 'light')
    assert.equal(output.dataset.defaultTheme, 'light')
    assert.equal(document.documentElement.classList.contains('light'), true)

    await act(async () => root.unmount())
    container.remove()
  })

  test('keeps an explicitly saved dark preference', async () => {
    document.cookie = 'vite-ui-theme=dark; path=/'
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <ThemeProvider>
          <ThemeState />
        </ThemeProvider>
      )
    })

    const output = container.querySelector('output')
    assert.ok(output)
    assert.equal(output.dataset.theme, 'dark')
    assert.equal(output.dataset.resolvedTheme, 'dark')
    assert.equal(document.documentElement.classList.contains('dark'), true)

    await act(async () => root.unmount())
    container.remove()
  })
})
