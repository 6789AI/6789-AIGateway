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
for (const key of [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
] as const) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Current Balance': 'Current Balance',
        'Remaining quota': 'Remaining quota',
        'Total Usage': 'Total Usage',
        'Total consumed quota': 'Total consumed quota',
        'Free Usage': 'Free Usage',
        'Active during promotions': 'Active during promotions',
        'API Requests': 'API Requests',
        'Total requests made': 'Total requests made',
      },
    },
  },
})

const { WalletStatsCard } = await import('../wallet-stats-card')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('wallet stats card', () => {
  after(() => {
    domWindow.close()
  })

  test('shows the remaining free usage and its promotion-only description', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <WalletStatsCard
            user={{
              id: 1,
              username: 'wallet-user',
              quota: 500000,
              used_quota: 1000000,
              request_count: 12,
              free_usage_limit: 20,
              free_usage_used: 7,
              free_usage_remaining: 13,
              free_usage_active: true,
              aff_quota: 0,
              aff_history_quota: 0,
              aff_count: 0,
              group: 'default',
            }}
          />
        </I18nextProvider>
      )
    })

    assert.equal(container.textContent?.includes('Free Usage'), true)
    assert.equal(container.textContent?.includes('13'), true)
    assert.equal(
      container.textContent?.includes('Active during promotions'),
      true
    )

    await act(async () => root.unmount())
    container.remove()
  })
})
