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
const { SidebarColorsSection } = await import('../sidebar-colors-section')

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

describe('sidebar color editor', () => {
  after(() => domWindow.close())

  test('rejects an invalid color, saves selected buttons, and restores defaults', async () => {
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
                <SidebarColorsSection config={{}} initialSerialized='{}' />
              </SettingsPageProvider>
            </I18nextProvider>
          </QueryClientProvider>
        )
      })

      const walletInput = container.querySelector<HTMLInputElement>(
        'input[name="/wallet"]'
      )
      const keysInput = container.querySelector<HTMLInputElement>(
        'input[name="/keys"]'
      )
      const saveButton = [...actionsContainer.querySelectorAll('button')].find(
        (button) => button.textContent?.includes('Save colors')
      )
      assert.ok(walletInput)
      assert.ok(keysInput)
      assert.ok(saveButton)

      await act(async () => {
        changeInputValue(walletInput, '#D57412')
        changeInputValue(keysInput, 'red')
      })
      await act(async () => saveButton.click())
      assert.equal(requests.length, 0)
      assert.match(container.textContent ?? '', /Invalid color/)

      await act(async () => {
        changeInputValue(keysInput, '#1256AB')
      })
      await act(async () => saveButton.click())
      assert.equal(requests.length, 1)
      assert.equal(requests[0].key, 'SidebarNavColors')
      assert.deepEqual(JSON.parse(requests[0].value), {
        '/keys': '#1256AB',
        '/wallet': '#D57412',
      })

      const savedColors = { '/keys': '#1256AB', '/wallet': '#D57412' }
      await act(async () => {
        root.render(
          <QueryClientProvider client={queryClient}>
            <I18nextProvider i18n={i18n}>
              <SettingsPageProvider actionsContainer={actionsContainer}>
                <SidebarColorsSection
                  config={savedColors}
                  initialSerialized={JSON.stringify(savedColors)}
                />
              </SettingsPageProvider>
            </I18nextProvider>
          </QueryClientProvider>
        )
      })
      const resetButton = [...actionsContainer.querySelectorAll('button')].find(
        (button) => button.textContent?.includes('Reset to default')
      )
      assert.ok(resetButton)
      await act(async () => resetButton.click())
      await act(async () => saveButton.click())
      assert.equal(requests.length, 2)
      assert.deepEqual(JSON.parse(requests[1].value), {})
    } finally {
      api.put = originalPut
      await act(async () => root.unmount())
      queryClient.clear()
      container.remove()
      actionsContainer.remove()
    }
  })
})
