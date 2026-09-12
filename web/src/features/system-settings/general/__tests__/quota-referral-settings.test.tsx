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
const { createMemoryHistory, createRootRoute, createRouter, RouterProvider } =
  await import('@tanstack/react-router')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { api } = await import('@/lib/api')
const { SettingsPageProvider } =
  await import('../../components/settings-page-context')
const { QuotaSettingsSection } = await import('../quota-settings-section')
const { quotaSchema } = await import('../quota-settings-schema')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type ApiPost = (url: string, body?: unknown) => Promise<{ data: unknown }>
const apiClient = api as unknown as { post: ApiPost }
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

describe('quota and referral settings', () => {
  after(() => {
    apiClient.post = originalPost
    domWindow.close()
  })

  test('shows independent referral controls and recalculates invitation counts', async () => {
    let requestedUrl = ''
    apiClient.post = async (url) => {
      requestedUrl = url
      return {
        data: {
          success: true,
          message: '',
          data: {
            users_scanned: 8,
            invitation_relations: 3,
            users_updated: 2,
          },
        },
      }
    }

    const container = document.createElement('div')
    const actionsContainer = document.createElement('div')
    document.body.append(container, actionsContainer)
    const root = createRoot(container)
    const queryClient = new QueryClient({
      defaultOptions: { mutations: { retry: false } },
    })
    const rootRoute = createRootRoute({
      component: () => (
        <SettingsPageProvider actionsContainer={actionsContainer}>
          <QuotaSettingsSection
            complianceConfirmed
            defaultValues={{
              QuotaForNewUser: 0,
              PreConsumedQuota: 500,
              QuotaForInviter: 10,
              QuotaForInvitee: 5,
              QuotaForInviterEnabled: true,
              QuotaForInviteeEnabled: true,
              AffiliateRebateEnabled: true,
              AffiliateRebatePercentage: 10,
              TopUpLink: '',
              general_setting: { docs_link: '' },
              quota_setting: { enable_free_model_pre_consume: true },
            }}
          />
        </SettingsPageProvider>
      ),
    })
    const router = createRouter({
      routeTree: rootRoute,
      history: createMemoryHistory({ initialEntries: ['/system-settings'] }),
    })

    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <I18nextProvider i18n={i18n}>
            <RouterProvider router={router} />
          </I18nextProvider>
        </QueryClientProvider>
      )
    })

    const referralTab = [...container.querySelectorAll('[role="tab"]')].find(
      (element) => element.textContent === 'Referrals'
    ) as HTMLButtonElement | undefined
    assert.ok(referralTab)
    await act(async () => referralTab.click())

    assert.equal(container.querySelectorAll('[role="switch"]').length, 3)
    const percentageInput = container.querySelector<HTMLInputElement>(
      'input[name="AffiliateRebatePercentage"]'
    )
    assert.ok(percentageInput)
    assert.equal(percentageInput.min, '0')
    assert.equal(percentageInput.max, '100')
    assert.equal(percentageInput.step, '0.01')

    const recalculateButton = [...container.querySelectorAll('button')].find(
      (button) => button.textContent?.includes('Recalculate counts')
    )
    assert.ok(recalculateButton)
    await act(async () => recalculateButton.click())

    const confirmButton = [...document.body.querySelectorAll('button')].find(
      (button) => button.textContent?.includes('Confirm recalculation')
    )
    assert.ok(confirmButton)
    await act(async () => confirmButton.click())
    await act(async () =>
      waitForCondition(
        () => container.textContent?.includes('Last result: 3') === true,
        'recalculation result was not rendered'
      )
    )

    assert.equal(
      requestedUrl,
      '/api/option/affiliate/recalculate-invite-counts'
    )

    await act(async () => root.unmount())
    queryClient.clear()
    container.remove()
    actionsContainer.remove()
  })

  test('validates rebate percentage range and precision', () => {
    const validSettings = {
      QuotaForNewUser: 0,
      PreConsumedQuota: 500,
      QuotaForInviter: 10,
      QuotaForInvitee: 5,
      QuotaForInviterEnabled: true,
      QuotaForInviteeEnabled: true,
      AffiliateRebateEnabled: true,
      AffiliateRebatePercentage: 10.25,
      TopUpLink: '',
      general_setting: { docs_link: '' },
      quota_setting: { enable_free_model_pre_consume: true },
    }

    assert.equal(quotaSchema.safeParse(validSettings).success, true)
    assert.equal(
      quotaSchema.safeParse({
        ...validSettings,
        AffiliateRebatePercentage: 10.001,
      }).success,
      false
    )
    assert.equal(
      quotaSchema.safeParse({
        ...validSettings,
        AffiliateRebatePercentage: 100.01,
      }).success,
      false
    )
  })
})
