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
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
  'scrollTo',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
} = await import('@tanstack/react-router')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { GlobalBannerFrame } = await import('../global-banner')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('global banner', () => {
  after(() => {
    domWindow.close()
  })

  test('renders one scheduled free-model banner with automatic content and countdown', async () => {
    const rootRoute = createRootRoute()
    const indexRoute = createRoute({
      getParentRoute: () => rootRoute,
      path: '/',
      component: () => (
        <I18nextProvider i18n={i18n}>
          <GlobalBannerFrame
            view={{
              visible: true,
              globalBanner: {
                enabled: false,
                content: '',
                background_color: '#0EA5E9',
                text_color: '#082F49',
                icon: 'gift',
                countdown_enabled: false,
                countdown_end_at: 0,
                link_url: '/special-pricing',
              },
              freeModelBanner: {
                background_color: '#123456',
                text_color: '#FEDCBA',
                icon: 'rocket',
                link_url: 'https://example.com/promotion',
              },
              models: [{ model_name: 'video-free' }],
              globalRemainingSeconds: null,
              promotionRemainingSeconds: 90_061,
            }}
          >
            <main>Route content</main>
          </GlobalBannerFrame>
        </I18nextProvider>
      ),
    })
    const pricingRoute = createRoute({
      getParentRoute: () => rootRoute,
      path: '/pricing',
      component: () => null,
    })
    const router = createRouter({
      routeTree: rootRoute.addChildren([indexRoute, pricingRoute]),
      history: createMemoryHistory({ initialEntries: ['/'] }),
    })
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(<RouterProvider router={router} />)
      await router.load()
    })

    const frame = container.querySelector('[data-global-banner-visible="true"]')
    const freeModelBanner = container.querySelector(
      'aside[aria-label="Limited-time free"]'
    )
    assert.ok(frame)
    assert.equal(frame.getAttribute('data-global-banner-count'), '1')
    assert.match(frame.getAttribute('style') ?? '', /padding-top: 2.5rem/i)
    assert.equal(
      document.documentElement.style.getPropertyValue('--global-banner-height'),
      '2.5rem'
    )
    assert.ok(freeModelBanner)
    assert.match(
      freeModelBanner.getAttribute('style') ?? '',
      /background-color: #123456/i
    )
    assert.match(freeModelBanner.getAttribute('style') ?? '', /color: #fedcba/i)
    assert.match(freeModelBanner.textContent ?? '', /video-free are free now/)
    assert.equal(freeModelBanner.querySelectorAll('svg').length, 2)
    assert.equal(
      freeModelBanner.querySelector('time')?.getAttribute('aria-label'),
      'Free period remaining: 01:01:01:01'
    )
    assert.equal(
      freeModelBanner
        .querySelector<HTMLAnchorElement>('a')
        ?.getAttribute('href'),
      'https://example.com/promotion'
    )
    assert.match(frame.textContent ?? '', /Route content/)

    await act(async () => root.unmount())
    container.remove()
  })

  test('does not reserve top space when no banner is visible', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <GlobalBannerFrame
          view={{
            visible: false,
            globalBanner: {
              enabled: false,
              content: '',
              background_color: '#0EA5E9',
              text_color: '#082F49',
              icon: 'gift',
              countdown_enabled: false,
              countdown_end_at: 0,
              link_url: '',
            },
            freeModelBanner: {
              background_color: '#A3E635',
              text_color: '#1A2E05',
              icon: 'megaphone',
              link_url: '',
            },
            models: [],
            globalRemainingSeconds: null,
            promotionRemainingSeconds: null,
          }}
        >
          <main>Route content</main>
        </GlobalBannerFrame>
      )
    })

    const frame = container.querySelector(
      '[data-global-banner-visible="false"]'
    )
    assert.ok(frame)
    assert.equal(frame.getAttribute('data-global-banner-count'), '0')
    assert.match(frame.getAttribute('style') ?? '', /padding-top: 0rem/i)
    assert.equal(container.querySelector('aside'), null)

    await act(async () => root.unmount())
    container.remove()
  })
})
