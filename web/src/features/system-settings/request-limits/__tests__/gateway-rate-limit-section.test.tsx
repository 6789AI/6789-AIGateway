/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import assert from 'node:assert/strict'
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window({ url: 'http://localhost/system-settings' })
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
  'HTMLInputElement',
  'HTMLFormElement',
  'HTMLLabelElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'PointerEvent',
  'MouseEvent',
  'FocusEvent',
  'CustomEvent',
  'MutationObserver',
  'ResizeObserver',
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
const { Toaster } = await import('sonner')
const { api } = await import('@/lib/api')
const { SettingsPageProvider } =
  await import('../../components/settings-page-context')
const { GatewayRateLimitSection } =
  await import('../gateway-rate-limit-section')
const { parseGatewayRateLimits } = await import('../gateway-rate-limit-schema')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type ApiPut = (url: string, body?: unknown) => Promise<{ data: unknown }>
const apiClient = api as unknown as { put: ApiPut }
const originalPut = apiClient.put

describe('website request limit settings', () => {
  after(() => {
    apiClient.put = originalPut
    domWindow.close()
  })

  test('shows five IP rules and loads saved values', async () => {
    const saved = parseGatewayRateLimits('')
    saved.api.limit = 800
    saved.session.enabled = true
    const container = document.createElement('div')
    const actionsContainer = document.createElement('div')
    document.body.append(container, actionsContainer)
    const root = createRoot(container)
    const queryClient = new QueryClient()

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <I18nextProvider i18n={i18n}>
            <SettingsPageProvider actionsContainer={actionsContainer}>
              <GatewayRateLimitSection defaultValue={JSON.stringify(saved)} />
            </SettingsPageProvider>
          </I18nextProvider>
        </QueryClientProvider>
      )
    })

    assert.equal(container.querySelectorAll('fieldset').length, 5)
    assert.equal(
      container.querySelector<HTMLInputElement>('input[name="api.limit"]')
        ?.value,
      '800'
    )
    assert.equal(
      container
        .querySelector('[aria-label="Enable Session refresh and logout limit"]')
        ?.getAttribute('aria-checked'),
      'true'
    )
    assert.ok(container.textContent?.includes('Save website request limits'))

    await act(async () => root.unmount())
    queryClient.clear()
    container.remove()
    actionsContainer.remove()
  })

  test('rejects invalid input without sending a setting update', async () => {
    let updateCount = 0
    apiClient.put = async () => {
      updateCount++
      return { data: { success: true, message: '' } }
    }
    const container = document.createElement('div')
    const actionsContainer = document.createElement('div')
    document.body.append(container, actionsContainer)
    const root = createRoot(container)
    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    })

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <I18nextProvider i18n={i18n}>
            <SettingsPageProvider actionsContainer={actionsContainer}>
              <GatewayRateLimitSection defaultValue='' />
            </SettingsPageProvider>
          </I18nextProvider>
        </QueryClientProvider>
      )
    })

    const limit = container.querySelector<HTMLInputElement>(
      'input[name="login.limit"]'
    )
    assert.ok(limit)
    await act(async () => {
      const valueSetter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        'value'
      )?.set
      assert.ok(valueSetter)
      valueSetter.call(limit, '0')
      limit.dispatchEvent(
        new domWindow.Event('input', { bubbles: true }) as unknown as Event
      )
      limit.dispatchEvent(
        new domWindow.Event('change', { bubbles: true }) as unknown as Event
      )
    })
    const save = container.querySelector<HTMLButtonElement>(
      'button[type="submit"]'
    )
    assert.ok(save)
    await act(async () => save.click())
    assert.equal(updateCount, 0)
    assert.equal(limit.getAttribute('aria-invalid'), 'true')

    await act(async () => root.unmount())
    queryClient.clear()
    container.remove()
    actionsContainer.remove()
  })

  test('shows a failed save and keeps the current form values', async () => {
    let updateCount = 0
    apiClient.put = async () => {
      updateCount++
      return { data: { success: false, message: 'server rejected limits' } }
    }
    const saved = parseGatewayRateLimits('')
    saved.api.limit = 801
    const container = document.createElement('div')
    const actionsContainer = document.createElement('div')
    document.body.append(container, actionsContainer)
    const root = createRoot(container)
    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    })

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <I18nextProvider i18n={i18n}>
            <SettingsPageProvider actionsContainer={actionsContainer}>
              <GatewayRateLimitSection defaultValue={JSON.stringify(saved)} />
            </SettingsPageProvider>
            <Toaster duration={60_000} />
          </I18nextProvider>
        </QueryClientProvider>
      )
    })

    const save = container.querySelector<HTMLButtonElement>(
      'button[type="submit"]'
    )
    assert.ok(save)
    await act(async () => save.click())
    assert.equal(updateCount, 1)
    assert.ok(document.body.textContent?.includes('server rejected limits'))
    assert.equal(
      container.querySelector<HTMLInputElement>('input[name="api.limit"]')
        ?.value,
      '801'
    )

    await act(async () => root.unmount())
    queryClient.clear()
    container.remove()
    actionsContainer.remove()
  })
})
