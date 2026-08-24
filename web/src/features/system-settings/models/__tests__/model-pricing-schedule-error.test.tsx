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

import type { ModelPricingEditorPanelHandle } from '../model-pricing-sheet'

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

describe('model pricing schedule errors', () => {
  after(() => {
    domWindow.close()
  })

  test('shows schedule validation errors while the price field is hidden', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    const panelRef = createRef<ModelPricingEditorPanelHandle>()

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ModelPricingEditorPanel
            ref={panelRef}
            editData={{
              name: 'token-model',
              ratio: '1',
              billingMode: 'per-token',
              priceSchedules: [
                {
                  id: 'invalid-range',
                  type: 'absolute',
                  adjustment_type: 'discount',
                  discount_rate: 0.5,
                  start_at: 200,
                  end_at: 100,
                },
              ],
            }}
          />
        </I18nextProvider>
      )
    })

    let result: unknown
    await act(async () => {
      result = await panelRef.current?.commitDraft()
    })

    assert.equal(result, null)
    assert.equal(
      container.querySelector('[data-slot="field-error"]')?.textContent,
      'The end time must be later than the start time.'
    )

    await act(async () => root.unmount())
    container.remove()
  })
})
