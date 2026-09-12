/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

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

const domWindow = new Window({ url: 'http://localhost/wallet' })
const domGlobals = [
  'window',
  'document',
  'navigator',
  'location',
  'history',
  'localStorage',
  'HTMLElement',
  'HTMLButtonElement',
  'HTMLInputElement',
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
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { api } = await import('@/lib/api')
const { useAuthStore } = await import('@/stores/auth-store')
const { AffiliateRewardsCard } = await import('../../affiliate-rewards-card')
const { BillingHistoryDialog } = await import('../billing-history-dialog')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type ApiGet = (url: string) => Promise<{ data: unknown }>
type ApiPost = (url: string, body?: unknown) => Promise<{ data: unknown }>
const apiClient = api as unknown as { get: ApiGet; post: ApiPost }
const originalGet = apiClient.get
const originalPost = apiClient.post

async function waitForCondition(
  condition: () => boolean,
  failureMessage: string
): Promise<void> {
  if (condition()) return
  await new Promise<void>((resolve, reject) => {
    const observer = new MutationObserver(() => {
      if (!condition()) return
      clearTimeout(timeoutId)
      observer.disconnect()
      resolve()
    })
    const timeoutId = setTimeout(() => {
      observer.disconnect()
      reject(new Error(`${failureMessage}: ${document.body.textContent}`))
    }, 1500)
    observer.observe(document, { childList: true, subtree: true })
  })
}

describe('manual top-up completion rebate choice', () => {
  after(() => {
    apiClient.get = originalGet
    apiClient.post = originalPost
    useAuthStore.getState().auth.reset()
    domWindow.close()
  })

  test('defaults to granting a rebate and submits an explicit opt-out', async () => {
    useAuthStore.getState().auth.setUser({ id: 1, username: 'root', role: 100 })
    apiClient.get = async () => ({
      data: {
        success: true,
        data: {
          items: [
            {
              id: 1,
              user_id: 2,
              amount: 1,
              money: 1,
              trade_no: 'PENDING-ORDER',
              payment_method: 'alipay',
              create_time: 1_700_000_000,
              status: 'pending',
            },
          ],
          total: 1,
        },
      },
    })
    let submittedBody: unknown
    apiClient.post = async (_url, body) => {
      submittedBody = body
      return {
        data: {
          success: true,
          data: {
            completed: true,
            credited_quota: 500_000,
            affiliate_rebate_granted: false,
            affiliate_rebate_quota: 0,
          },
        },
      }
    }

    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <BillingHistoryDialog open onOpenChange={() => undefined} />
        </I18nextProvider>
      )
    })
    await act(async () =>
      waitForCondition(
        () => document.body.textContent?.includes('PENDING-ORDER') === true,
        'pending order was not rendered'
      )
    )

    const completeButton = [...document.body.querySelectorAll('button')].find(
      (button) => button.textContent === 'Complete Order'
    )
    assert.ok(completeButton)
    await act(async () => completeButton.click())

    const rebateSwitch =
      document.querySelector<HTMLButtonElement>('[role="switch"]')
    assert.ok(rebateSwitch)
    assert.equal(rebateSwitch.hasAttribute('data-checked'), true)
    await act(async () => rebateSwitch.click())
    assert.equal(rebateSwitch.hasAttribute('data-unchecked'), true)

    const confirmButton = [...document.body.querySelectorAll('button')].find(
      (button) => button.textContent === 'Confirm'
    )
    assert.ok(confirmButton)
    await act(async () => confirmButton.click())
    await act(async () =>
      waitForCondition(
        () => submittedBody !== undefined,
        'completion request was not submitted'
      )
    )
    await act(async () =>
      waitForCondition(
        () => document.querySelector('[role="alertdialog"]') === null,
        'completion dialog did not close'
      )
    )

    assert.deepEqual(submittedBody, {
      trade_no: 'PENDING-ORDER',
      grant_affiliate_rebate: false,
    })

    await act(async () => root.unmount())
    container.remove()
  })

  test('labels untransferred affiliate quota as pending transfer rewards', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <AffiliateRewardsCard
            user={{
              id: 1,
              username: 'inviter',
              quota: 0,
              used_quota: 0,
              request_count: 0,
              free_usage_limit: 0,
              free_usage_used: 0,
              free_usage_remaining: 0,
              free_usage_active: false,
              aff_quota: 100,
              aff_history_quota: 200,
              aff_count: 2,
              group: 'default',
            }}
            affiliateLink='http://localhost/sign-up?aff=TEST'
            onTransfer={() => undefined}
          />
        </I18nextProvider>
      )
    })

    assert.match(container.textContent ?? '', /Pending Transfer Rewards/)

    await act(async () => root.unmount())
    container.remove()
  })
})
