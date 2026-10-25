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

import type {
  ModelPricingEditorPanelHandle,
  ModelRatioData,
} from '../model-pricing-sheet'

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

const { act, createRef } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { ModelPricingEditorPanel }: typeof import('../model-pricing-sheet') =
  await import('../model-pricing-sheet')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('video quality pricing editor', () => {
  after(() => domWindow.close())
  test('loads five tiers, saves edits and free prices, rejects invalid values, and clears a tier to fallback', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    const panelRef = createRef<ModelPricingEditorPanelHandle>()
    try {
      await act(async () =>
        root.render(
          <I18nextProvider i18n={i18n}>
            <ModelPricingEditorPanel
              ref={panelRef}
              editData={{
                name: 'video',
                price: '0.5',
                billingMode: 'per-request',
                multiplyByDuration: false,
                videoResolutionPrices: {
                  '480p': 0,
                  '720p': 0.1,
                  '768p': 0.2,
                  '1080p': 0.3,
                  '2k': 0.4,
                },
              }}
            />
          </I18nextProvider>
        )
      )
      for (const labelText of ['480p', '720p', '768p', '1080p', '2K']) {
        const label = [...container.querySelectorAll('label')].find(
          (label) => label.textContent === labelText
        )
        assert.ok(label?.control, `missing accessible input ${labelText}`)
      }
      const label = [...container.querySelectorAll('label')].find(
        (label) => label.textContent === '768p'
      )
      const input = label?.control as HTMLInputElement
      assert.equal(input.value, '0.2')
      const setter = Object.getOwnPropertyDescriptor(
        HTMLInputElement.prototype,
        'value'
      )?.set
      assert.ok(setter)
      await act(async () => {
        setter.call(input, '0.75')
        input.dispatchEvent(new Event('input', { bubbles: true }))
      })
      let result: ModelRatioData | null | undefined
      await act(async () => {
        result = await panelRef.current?.commitDraft()
      })
      assert.deepEqual(result?.videoResolutionPrices, {
        '480p': 0,
        '720p': 0.1,
        '768p': 0.75,
        '1080p': 0.3,
        '2k': 0.4,
      })
      await act(async () => {
        setter.call(input, '-1')
        input.dispatchEvent(new Event('input', { bubbles: true }))
      })
      await act(async () => {
        result = await panelRef.current?.commitDraft()
      })
      assert.equal(result === null, true)
      assert.equal(input.getAttribute('aria-invalid'), 'true')
      assert.ok(
        container.querySelector('[data-slot="field-error"]')?.textContent
      )
      await act(async () => {
        setter.call(input, '')
        input.dispatchEvent(new Event('input', { bubbles: true }))
      })
      await act(async () => {
        result = await panelRef.current?.commitDraft()
      })
      assert.equal(result?.videoResolutionPrices?.['768p'], undefined)
      assert.equal(result?.videoResolutionPrices?.['480p'], 0)
      const durationLabel = [...container.querySelectorAll('label')].find(
        (item) => item.textContent === 'Multiply fixed video price by duration'
      )
      assert.ok(durationLabel?.control)
      await act(async () => durationLabel.control?.click())
      const fieldset = input.closest('fieldset')
      assert.ok(fieldset?.textContent?.includes('per second'))
      await act(async () => {
        result = await panelRef.current?.commitDraft()
      })
      assert.equal(result?.multiplyByDuration, true)
    } finally {
      await act(async () => root.unmount())
      container.remove()
    }
  })
})
