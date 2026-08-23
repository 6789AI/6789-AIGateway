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

const domWindow = new Window()
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
const { NoticeSection } = await import('../notice-section')

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

describe('marketing banner settings', () => {
  after(() => {
    domWindow.close()
  })

  test('requires content when enabled and previews custom colors', async () => {
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
              <NoticeSection
                defaultValues={{
                  Notice: '',
                  MarketingBannerEnabled: false,
                  MarketingBannerContent: '',
                  MarketingBannerBackgroundColor: '#A3E635',
                  MarketingBannerTextColor: '#1A2E05',
                  MarketingBannerIcon: 'megaphone',
                  MarketingBannerCountdownEnabled: false,
                  MarketingBannerCountdownEndAt: 0,
                  MarketingBannerLinkURL: '',
                }}
              />
            </SettingsPageProvider>
          </I18nextProvider>
        </QueryClientProvider>
      )
    })

    const bannerSwitch = container.querySelector<HTMLElement>('[role="switch"]')
    assert.ok(bannerSwitch)
    await act(async () => bannerSwitch.click())

    const saveButton = [...actionsContainer.querySelectorAll('button')].find(
      (button) => button.textContent === 'Save notice and banner'
    )
    assert.ok(saveButton)
    await act(async () => saveButton.click())
    assert.match(
      container.textContent ?? '',
      /Content is required when the marketing banner is enabled/
    )

    const contentInput = container.querySelector<HTMLInputElement>(
      'input[name="MarketingBannerContent"]'
    )
    const backgroundInput = container.querySelector<HTMLInputElement>(
      'input[name="MarketingBannerBackgroundColor"]'
    )
    const textInput = container.querySelector<HTMLInputElement>(
      'input[name="MarketingBannerTextColor"]'
    )
    const linkInput = container.querySelector<HTMLInputElement>(
      'input[name="MarketingBannerLinkURL"]'
    )
    const couponButton = container.querySelector<HTMLElement>(
      '[aria-label="Coupon"]'
    )
    assert.ok(contentInput)
    assert.ok(backgroundInput)
    assert.ok(textInput)
    assert.ok(linkInput)
    assert.ok(couponButton)

    await act(async () => {
      changeInputValue(contentInput, 'Weekend bonus')
      changeInputValue(backgroundInput, '#123456')
      changeInputValue(textInput, '#FEDCBA')
      changeInputValue(linkInput, '/wallet')
      couponButton.click()
    })
    assert.equal(linkInput.checkValidity(), true)

    const preview = container.querySelector('[aria-label="Banner preview"]')
    assert.ok(preview)
    assert.match(preview.textContent ?? '', /Weekend bonus/)
    assert.match(
      preview.getAttribute('style') ?? '',
      /background-color: #123456/i
    )
    assert.match(preview.getAttribute('style') ?? '', /color: #fedcba/i)
    assert.equal(couponButton.getAttribute('aria-pressed'), 'true')

    const switches = container.querySelectorAll<HTMLElement>('[role="switch"]')
    assert.equal(switches.length, 2)
    await act(async () => switches[1].click())
    const countdownInput = container.querySelector<HTMLInputElement>(
      'input[type="datetime-local"]'
    )
    assert.ok(countdownInput)
    await act(async () => {
      changeInputValue(countdownInput, '2099-01-01T00:00')
    })
    assert.match(preview.textContent ?? '', /\d{2,}:\d{2}:\d{2}:\d{2}/)

    await act(async () => root.unmount())
    container.remove()
    actionsContainer.remove()
    queryClient.clear()
  })
})
