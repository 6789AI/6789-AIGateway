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
  'HTMLInputElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { api } = await import('@/lib/api')
const { SettingsPageProvider } =
  await import('../../components/settings-page-context')
const { HEADER_NAV_DEFAULT, serializeHeaderNavModules } =
  await import('../config')
const { HeaderNavigationSection } = await import('../header-navigation-section')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

function changeInputValue(input: HTMLInputElement, value: string) {
  const valueSetter = Object.getOwnPropertyDescriptor(
    domWindow.HTMLInputElement.prototype,
    'value'
  )?.set
  assert.ok(valueSetter)
  valueSetter.call(input, value)
  input.dispatchEvent(
    new domWindow.Event('input', { bubbles: true }) as unknown as Event
  )
}

describe('header navigation editor', () => {
  after(() => domWindow.close())

  test('adds and reorders a link, rejects an unsafe URL, then saves the ordered list', async () => {
    const requests: Array<{ key: string; value: string }> = []
    const originalPut = api.put
    api.put = (async (_url: string, body: { key: string; value: string }) => {
      requests.push(body)
      return { data: { success: true } }
    }) as typeof api.put

    const container = document.createElement('div')
    const actionsContainer = document.createElement('div')
    document.body.append(container, actionsContainer)
    const root = createRoot(container)
    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    })

    try {
      await act(async () => {
        root.render(
          <QueryClientProvider client={queryClient}>
            <I18nextProvider i18n={i18n}>
              <SettingsPageProvider actionsContainer={actionsContainer}>
                <HeaderNavigationSection
                  config={HEADER_NAV_DEFAULT}
                  initialSerialized={serializeHeaderNavModules(
                    HEADER_NAV_DEFAULT
                  )}
                />
              </SettingsPageProvider>
            </I18nextProvider>
          </QueryClientProvider>
        )
      })

      const addButton = [...container.querySelectorAll('button')].find(
        (button) => button.textContent?.includes('Add link')
      )
      assert.ok(addButton)
      await act(async () => {
        addButton.click()
      })

      const nameInput = container.querySelector<HTMLInputElement>(
        'input[name="links.7.label"]'
      )
      const urlInput = container.querySelector<HTMLInputElement>(
        'input[name="links.7.href"]'
      )
      assert.ok(nameInput)
      assert.ok(urlInput)
      await act(async () => {
        changeInputValue(nameInput, 'Support')
        changeInputValue(urlInput, 'javascript:alert(1)')
      })

      const saveButton = [...actionsContainer.querySelectorAll('button')].find(
        (button) => button.textContent?.includes('Save navigation')
      )
      assert.ok(saveButton)
      await act(async () => {
        saveButton.click()
      })
      assert.equal(requests.length, 0)
      assert.match(container.textContent ?? '', /Invalid navigation URL/)

      await act(async () => {
        changeInputValue(urlInput, 'https://example.com/support')
      })
      const customRow = urlInput.closest('.space-y-3.py-3')
      const moveUp = customRow?.querySelector<HTMLButtonElement>(
        'button[aria-label="Move up"]'
      )
      assert.ok(moveUp)
      await act(async () => {
        moveUp.click()
      })
      await act(async () => {
        saveButton.click()
      })

      assert.equal(requests.length, 1)
      assert.equal(requests[0].key, 'HeaderNavModules')
      const saved = JSON.parse(requests[0].value) as {
        links: Array<{ key: string; label: string; href: string }>
      }
      assert.equal(saved.links[6].label, 'Support')
      assert.equal(saved.links[6].href, 'https://example.com/support')
    } finally {
      api.put = originalPut
      await act(async () => root.unmount())
      queryClient.clear()
      container.remove()
      actionsContainer.remove()
    }
  })
})
