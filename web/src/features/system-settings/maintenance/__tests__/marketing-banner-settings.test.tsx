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

describe('global and free-model banner settings', () => {
  after(() => {
    domWindow.close()
  })

  test('shows only style controls for automatic free-model promotions', async () => {
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
                  GlobalBannerEnabled: false,
                  GlobalBannerContent: '',
                  GlobalBannerBackgroundColor: '#0EA5E9',
                  GlobalBannerTextColor: '#082F49',
                  GlobalBannerIcon: 'gift',
                  GlobalBannerCountdownEnabled: false,
                  GlobalBannerCountdownEndAt: 0,
                  GlobalBannerLinkURL: '',
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
    assert.match(container.textContent ?? '', /Global announcement banner/)
    assert.match(container.textContent ?? '', /Limited-time model offer/)
    assert.equal(
      container.querySelectorAll('[aria-label="Banner preview"]').length,
      2
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
    const freeModelHeading = [...container.querySelectorAll('h3')].find(
      (heading) => heading.textContent === 'Limited-time model offer'
    )
    const freeModelSection = freeModelHeading?.closest('section')
    const couponButton = freeModelSection?.querySelector<HTMLElement>(
      '[aria-label="Coupon"]'
    )
    assert.ok(backgroundInput)
    assert.ok(textInput)
    assert.ok(linkInput)
    assert.ok(freeModelSection)
    assert.ok(couponButton)
    const customIconInputs = container.querySelectorAll<HTMLInputElement>(
      'input[aria-label="Custom characters or emoji"]'
    )
    assert.equal(customIconInputs.length, 2)
    assert.equal(
      freeModelSection.querySelector('input[name="MarketingBannerContent"]'),
      null
    )
    assert.equal(freeModelSection.querySelector('[role="switch"]'), null)
    assert.equal(
      freeModelSection.querySelector('input[type="datetime-local"]'),
      null
    )

    await act(async () => {
      changeInputValue(backgroundInput, '#123456')
      changeInputValue(textInput, '#FEDCBA')
      changeInputValue(linkInput, '/wallet')
      couponButton.click()
    })
    assert.equal(linkInput.checkValidity(), true)

    const preview = freeModelSection.querySelector(
      '[aria-label="Banner preview"]'
    )
    assert.ok(preview)
    assert.match(preview.textContent ?? '', /Model A is 20% off/)
    assert.match(preview.textContent ?? '', /00:01:23:45/)
    assert.match(
      preview.getAttribute('style') ?? '',
      /background-color: #123456/i
    )
    assert.match(preview.getAttribute('style') ?? '', /color: #fedcba/i)
    assert.equal(couponButton.getAttribute('aria-pressed'), 'true')
    const iconCount = preview.querySelectorAll('svg').length
    await act(async () => couponButton.click())
    assert.equal(couponButton.getAttribute('aria-pressed'), 'false')
    assert.equal(preview.querySelectorAll('svg').length, iconCount - 1)

    await act(async () => {
      changeInputValue(customIconInputs[1], '🎉 限时优惠')
    })
    assert.equal(couponButton.getAttribute('aria-pressed'), 'false')
    assert.match(preview.textContent ?? '', /🎉 限时优惠/)
    assert.equal(
      preview.querySelector('[data-banner-custom-icon]')?.textContent,
      '🎉 限时优惠'
    )

    await act(async () => {
      changeInputValue(customIconInputs[0], '📣 公告')
      changeInputValue(customIconInputs[1], '🎉'.repeat(101))
    })
    assert.equal(
      [...customIconInputs[1].value].length,
      100,
      'emoji must be counted by Unicode code point'
    )
    const previews = container.querySelectorAll<HTMLElement>(
      '[aria-label="Banner preview"]'
    )
    assert.equal(
      previews[0]?.querySelector('[data-banner-custom-icon]')?.textContent,
      '📣 公告'
    )

    assert.equal(
      container.querySelectorAll<HTMLElement>('[role="switch"]').length,
      2
    )

    await act(async () => root.unmount())
    container.remove()
    actionsContainer.remove()
    queryClient.clear()
  })
})
