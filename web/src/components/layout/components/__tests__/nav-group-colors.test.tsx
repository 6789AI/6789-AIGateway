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
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window({ url: 'http://localhost/' })
for (const key of [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
] as const) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { RouterProvider, createMemoryHistory, createRootRoute, createRouter } =
  await import('@tanstack/react-router')
const { NavGroup } = await import('../nav-group')
const { SidebarProvider } = await import('@/components/ui/sidebar')

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('sidebar navigation colors', () => {
  after(() => domWindow.close())

  test('colors the selected button and leaves unconfigured buttons on the theme color', async () => {
    const rootRoute = createRootRoute({
      component: () => (
        <SidebarProvider>
          <NavGroup
            title='General'
            items={[
              { title: 'Wallet', url: '/wallet' },
              { title: 'API Keys', url: '/keys' },
            ]}
            colors={{ '/wallet': '#D57412' }}
          />
        </SidebarProvider>
      ),
    })
    const router = createRouter({
      routeTree: rootRoute,
      history: createMemoryHistory({ initialEntries: ['/'] }),
    })
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    try {
      await act(async () => {
        root.render(<RouterProvider router={router} />)
        await router.load()
      })

      const wallet = [...container.querySelectorAll('a')].find(
        (link) => link.textContent === 'Wallet'
      )
      const keys = [...container.querySelectorAll('a')].find(
        (link) => link.textContent === 'API Keys'
      )
      assert.ok(wallet)
      assert.ok(keys)
      assert.equal(wallet.style.color, '#D57412')
      assert.equal(keys.style.color, '')
    } finally {
      await act(async () => root.unmount())
      container.remove()
    }
  })
})
