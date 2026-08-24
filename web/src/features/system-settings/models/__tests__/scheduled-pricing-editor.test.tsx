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

import type { PriceSchedule } from '../model-pricing-core'

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
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { ScheduledPricingEditor } = await import('../scheduled-pricing-editor')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('scheduled pricing editor', () => {
  after(() => {
    domWindow.close()
  })

  test('shows a free fixed range and adds another schedule', async () => {
    const initial: PriceSchedule[] = [
      {
        id: 'fixed',
        type: 'absolute',
        price: 0,
        start_at: 1_800_000_000,
        end_at: 1_800_086_400,
      },
    ]
    let changed: PriceSchedule[] | undefined
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ScheduledPricingEditor
            schedules={initial}
            allowFixedPrice
            onChange={(schedules) => {
              changed = schedules
            }}
          />
        </I18nextProvider>
      )
    })

    assert.equal(
      container.querySelector<HTMLInputElement>('input[type="number"]')?.value,
      '0'
    )
    assert.equal(
      container.querySelectorAll<HTMLInputElement>(
        'input[type="datetime-local"]'
      ).length,
      2
    )

    const bannerSwitch = container.querySelector<HTMLElement>('[role="switch"]')
    assert.ok(bannerSwitch)
    assert.equal(bannerSwitch.getAttribute('aria-checked'), 'true')
    await act(async () => bannerSwitch.click())
    assert.equal(changed?.[0]?.show_banner, false)

    const addButton = [...container.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('Add schedule')
    )
    assert.ok(addButton)
    await act(async () => addButton.click())
    assert.equal(changed?.length, 2)
    assert.equal(changed?.[1]?.price, 0)
    assert.equal(changed?.[1]?.show_banner, true)

    await act(async () => root.unmount())
    container.remove()
  })

  test('shows selected weekdays and an overnight weekly range', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ScheduledPricingEditor
            schedules={[
              {
                id: 'weekly',
                type: 'weekly',
                price: 0.1,
                weekdays: [1, 5],
                start_minute: 22 * 60,
                end_minute: 2 * 60,
                timezone: 'Asia/Shanghai',
              },
            ]}
            allowFixedPrice
            onChange={() => {}}
          />
        </I18nextProvider>
      )
    })

    const weekdayButtons = [
      ...container.querySelectorAll<HTMLButtonElement>(
        'button[aria-label="Sunday"], button[aria-label="Monday"], button[aria-label="Tuesday"], button[aria-label="Wednesday"], button[aria-label="Thursday"], button[aria-label="Friday"], button[aria-label="Saturday"]'
      ),
    ]
    assert.equal(weekdayButtons.length, 7)
    assert.equal(
      weekdayButtons.filter((button) => button.dataset.pressed === '').length,
      2
    )
    assert.deepEqual(
      [
        ...container.querySelectorAll<HTMLInputElement>('input[type="time"]'),
      ].map((input) => input.value),
      ['22:00', '02:00']
    )

    await act(async () => root.unmount())
    container.remove()
  })

  test('renders when i18next uses the internal Chinese locale code', async () => {
    const chineseI18n = createInstance()
    await chineseI18n.use(initReactI18next).init({
      fallbackLng: false,
      lng: 'zhCN',
      resources: { zhCN: { translation: {} } },
    })
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={chineseI18n}>
          <ScheduledPricingEditor
            schedules={[]}
            allowFixedPrice
            onChange={() => {}}
          />
        </I18nextProvider>
      )
    })

    assert.ok(
      [...container.querySelectorAll('button')].some((button) =>
        button.textContent?.includes('Add schedule')
      )
    )

    await act(async () => root.unmount())
    container.remove()
  })

  test('adds a discount activity when fixed prices are unavailable', async () => {
    let changed: PriceSchedule[] | undefined
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ScheduledPricingEditor
            schedules={[]}
            allowFixedPrice={false}
            onChange={(schedules) => {
              changed = schedules
            }}
          />
        </I18nextProvider>
      )
    })

    const addButton = [...container.querySelectorAll('button')].find((button) =>
      button.textContent?.includes('Add schedule')
    )
    assert.ok(addButton)
    await act(async () => addButton.click())
    assert.equal(changed?.[0]?.adjustment_type, 'discount')
    assert.equal(changed?.[0]?.discount_rate, 0)
    assert.equal(changed?.[0]?.price, undefined)

    await act(async () => root.unmount())
    container.remove()
  })

  test('associates price and discount labels with their numeric inputs', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ScheduledPricingEditor
            schedules={[
              {
                id: 'fixed-price',
                type: 'absolute',
                adjustment_type: 'fixed_price',
                price: 0.1,
                start_at: 100,
                end_at: 200,
              },
              {
                id: 'discount-price',
                type: 'absolute',
                adjustment_type: 'discount',
                discount_rate: 0.8,
                start_at: 300,
                end_at: 400,
              },
            ]}
            allowFixedPrice
            onChange={() => {}}
          />
        </I18nextProvider>
      )
    })

    const labels = [...container.querySelectorAll('label')]
    const fixedLabel = labels.find((label) =>
      label.textContent?.includes('Activity price')
    )
    const discountLabel = labels.find((label) =>
      label.textContent?.includes('Discounted price percentage')
    )
    assert.equal(fixedLabel?.htmlFor, 'fixed-price-activity-price')
    assert.equal(discountLabel?.htmlFor, 'discount-price-discount-rate')
    assert.ok(container.querySelector('#fixed-price-activity-price'))
    assert.ok(container.querySelector('#discount-price-discount-rate'))

    await act(async () => root.unmount())
    container.remove()
  })
})
