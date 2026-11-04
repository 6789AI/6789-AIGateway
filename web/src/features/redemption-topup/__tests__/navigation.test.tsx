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
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { NavGroup } = await import('@/components/layout/components/nav-group')
const { SidebarProvider } = await import('@/components/ui/sidebar')
const { useSidebarData } = await import('@/hooks/use-sidebar-data')
const { RedemptionTopup } = await import('../index')
const { redemptionTopupShopUrl } = await import('../constants')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

function PersonalNavigation() {
  const personal = useSidebarData().navGroups.find(
    (group) => group.id === 'personal'
  )
  assert.ok(personal)
  return (
    <SidebarProvider>
      <NavGroup {...personal} />
    </SidebarProvider>
  )
}

describe('redemption top-up navigation', () => {
  after(() => domWindow.close())

  test('sidebar opens the shop as a top-level tab', async () => {
    const rootRoute = createRootRoute({
      component: () => (
        <I18nextProvider i18n={i18n}>
          <PersonalNavigation />
        </I18nextProvider>
      ),
    })
    const router = createRouter({
      routeTree: rootRoute,
      history: createMemoryHistory({ initialEntries: ['/'] }),
    })
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(<RouterProvider router={router} />)
      await router.load()
    })

    const link = [...container.querySelectorAll('a')].find((anchor) =>
      anchor.textContent?.includes('Redemption Code Top-up')
    )
    assert.ok(link)
    assert.equal(link.href, redemptionTopupShopUrl)
    assert.equal(link.target, '_blank')
    assert.match(link.rel, /noopener/)
    assert.equal(container.querySelector('iframe'), null)

    await act(async () => root.unmount())
    container.remove()
  })

  test('direct route offers the same external link without an iframe', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <RedemptionTopup />
        </I18nextProvider>
      )
    })

    const link = container.querySelector<HTMLAnchorElement>('main a')
    assert.ok(link)
    assert.equal(link.href, redemptionTopupShopUrl)
    assert.equal(link.target, '_blank')
    assert.match(link.rel, /noopener/)
    assert.equal(container.querySelector('iframe'), null)

    await act(async () => root.unmount())
    container.remove()
  })
})
