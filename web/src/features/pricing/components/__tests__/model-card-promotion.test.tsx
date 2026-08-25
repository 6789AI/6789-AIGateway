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

import type { ModelPromotion } from '@/features/global-banner'

import type { PricingModel } from '../../types'

const domWindow = new Window({ url: 'http://localhost/pricing' })
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'matchMedia',
  'customElements',
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
const { ModelCard } = await import('../model-card')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Free offer': 'Free',
        '{{percent}}% off': '{{percent}}% off',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const model: PricingModel = {
  id: 1,
  model_name: 'promotion-model',
  description: 'Promotion model',
  quota_type: 1,
  model_ratio: 1,
  completion_ratio: 1,
  model_price: 0.1,
  enable_groups: ['default'],
}

async function renderCard(promotion?: ModelPromotion) {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <ModelCard
          model={{ ...model, active_promotion: promotion }}
          onClick={() => {}}
        />
      </I18nextProvider>
    )
  })

  return { container, root }
}

describe('model card promotions', () => {
  after(() => {
    domWindow.close()
  })

  test('highlights a free model and shows a free badge', async () => {
    const rendered = await renderCard({
      model_name: model.model_name,
      promotion_type: 'free',
    })

    const card =
      rendered.container.querySelector<HTMLElement>('[data-model-card]')
    const badge = rendered.container.querySelector<HTMLElement>(
      '[data-model-promotion-badge]'
    )
    assert.ok(card)
    assert.equal(card.dataset.modelPromotion, 'free')
    assert.ok(card.classList.contains('bg-promotion/10'))
    assert.equal(badge?.textContent, 'Free')

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })

  test('shows the saved percentage for a discounted model', async () => {
    const rendered = await renderCard({
      model_name: model.model_name,
      promotion_type: 'discount',
      discount_rate: 0.8,
    })

    const card =
      rendered.container.querySelector<HTMLElement>('[data-model-card]')
    const badge = rendered.container.querySelector<HTMLElement>(
      '[data-model-promotion-badge]'
    )
    assert.ok(card)
    assert.equal(card.dataset.modelPromotion, 'discount')
    assert.ok(card.classList.contains('bg-promotion/10'))
    assert.equal(badge?.textContent, '20% off')

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })

  test('shows a special-price badge for a fixed-price activity', async () => {
    const rendered = await renderCard({
      model_name: model.model_name,
      promotion_type: 'fixed_price',
      price: 0.05,
    })

    const card =
      rendered.container.querySelector<HTMLElement>('[data-model-card]')
    const badge = rendered.container.querySelector<HTMLElement>(
      '[data-model-promotion-badge]'
    )
    assert.ok(card)
    assert.equal(card.dataset.modelPromotion, 'fixed_price')
    assert.ok(card.classList.contains('bg-promotion/10'))
    assert.equal(badge?.textContent, 'Special price')

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })

  test('keeps the standard card appearance without an active promotion', async () => {
    const rendered = await renderCard()

    const card =
      rendered.container.querySelector<HTMLElement>('[data-model-card]')
    assert.ok(card)
    assert.equal(card.hasAttribute('data-model-promotion'), false)
    assert.ok(!card.classList.contains('bg-promotion/10'))
    assert.equal(
      rendered.container.querySelector('[data-model-promotion-badge]'),
      null
    )

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })
})
