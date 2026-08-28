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
const { SettingsPageProvider } =
  await import('../../components/settings-page-context')
const { BotProtectionSection } = await import('../bot-protection-section')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const defaultValues = {
  TurnstileCheckEnabled: false,
  BotProtectionProvider: 'turnstile' as const,
  BotProtectionLoginEnabled: true,
  BotProtectionRegisterEnabled: true,
  BotProtectionEmailVerificationEnabled: true,
  BotProtectionPasswordResetEnabled: true,
  BotProtectionCheckinEnabled: true,
  TurnstileSiteKey: '',
  TurnstileSecretKey: '',
  ReCaptchaSiteKey: '',
  ReCaptchaSecretKey: '',
  GeeTestCaptchaId: '',
  GeeTestSecretKey: '',
  CapServerURL: '',
  CapSiteKey: '',
  CapSecretKey: '',
}

describe('bot protection settings', () => {
  after(() => domWindow.close())

  test('switches provider settings and exposes independent feature toggles', async () => {
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
              <BotProtectionSection
                defaultValues={defaultValues}
                secretConfigured={{
                  turnstile: false,
                  recaptcha: false,
                  geetest_v4: false,
                  cap: false,
                }}
              />
            </SettingsPageProvider>
          </I18nextProvider>
        </QueryClientProvider>
      )
    })

    const providerButtons = [
      ...container.querySelectorAll<HTMLButtonElement>('[role="radio"]'),
    ]
    assert.equal(providerButtons.length, 4)
    assert.equal(providerButtons[0]?.getAttribute('aria-checked'), 'true')
    assert.equal(container.querySelectorAll('[role="switch"]').length, 6)
    assert.equal(
      container.querySelector('input[name="TurnstileSiteKey"]') !== null,
      true
    )

    const capButton = providerButtons.find(
      (button) => button.textContent === 'Cap'
    )
    assert.ok(capButton)
    await act(async () => {
      capButton.dispatchEvent(
        new domWindow.MouseEvent('click', { bubbles: true }) as unknown as Event
      )
    })

    assert.equal(capButton.getAttribute('aria-checked'), 'true')
    assert.equal(
      container.querySelector('input[name="CapServerURL"]') !== null,
      true
    )
    assert.equal(
      container.querySelector('input[name="TurnstileSiteKey"]'),
      null
    )

    await act(async () => root.unmount())
    queryClient.clear()
    container.remove()
    actionsContainer.remove()
  })
})
