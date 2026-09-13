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
import { after, afterEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window({ url: 'http://localhost/users' })
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
  'HTMLFormElement',
  'HTMLLabelElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'KeyboardEvent',
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

const { act, useState } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { api } = await import('@/lib/api')
const { InvitationLookupDialog } = await import('../invitation-lookup-dialog')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type ApiGet = (
  url: string,
  config?: { params?: Record<string, unknown> }
) => Promise<{ data: unknown }>
type MockableApi = { get: ApiGet }
type RenderedDialog = {
  host: HTMLDivElement
  queryClient: InstanceType<typeof QueryClient>
  root: ReturnType<typeof createRoot>
}

const apiClient = api as unknown as MockableApi
const originalGet = apiClient.get
let renderedDialog: RenderedDialog | null = null

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((promiseResolve) => {
    resolve = promiseResolve
  })
  return { promise, resolve }
}

function user(id: number, username: string, deletedAt: string | null = null) {
  return {
    id,
    username,
    display_name: `${username} display`,
    email: `${username}@example.com`,
    status: 1,
    created_at: 1_700_000_000 + id,
    deleted_at: deletedAt,
  }
}

function DialogHarness() {
  const [open, setOpen] = useState(true)
  return (
    <>
      <button id='reopen-dialog' type='button' onClick={() => setOpen(true)}>
        Reopen
      </button>
      <InvitationLookupDialog open={open} onOpenChange={setOpen} />
    </>
  )
}

async function renderDialog(): Promise<void> {
  const host = document.createElement('div')
  document.body.append(host)
  const root = createRoot(host)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  renderedDialog = { host, queryClient, root }

  await act(async () =>
    root.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <DialogHarness />
        </I18nextProvider>
      </QueryClientProvider>
    )
  )
}

async function waitForCondition(
  condition: () => boolean,
  failureMessage: string
): Promise<void> {
  if (condition()) return

  await new Promise<void>((resolve, reject) => {
    const finishIfReady = () => {
      if (!condition()) return
      clearTimeout(timeoutId)
      observer.disconnect()
      resolve()
    }
    const observer = new MutationObserver(() => queueMicrotask(finishIfReady))
    const timeoutId = setTimeout(() => {
      observer.disconnect()
      if (condition()) {
        resolve()
        return
      }
      reject(new Error(`${failureMessage}: ${document.body.textContent}`))
    }, 1500)

    observer.observe(document, {
      attributes: true,
      childList: true,
      characterData: true,
      subtree: true,
    })
  })
}

function getLookupInput(): HTMLInputElement {
  const input = document.querySelector<HTMLInputElement>(
    '#affiliate-lookup-query'
  )
  assert.ok(input)
  return input
}

async function changeInput(value: string): Promise<void> {
  const input = getLookupInput()
  await act(async () => {
    const valueSetter = Object.getOwnPropertyDescriptor(
      domWindow.HTMLInputElement.prototype,
      'value'
    )?.set
    assert.ok(valueSetter)
    valueSetter.call(input, value)
    input.dispatchEvent(
      new domWindow.Event('input', { bubbles: true }) as unknown as Event
    )
  })
}

async function submitLookup(): Promise<void> {
  const form = document.querySelector<HTMLFormElement>('form')
  assert.ok(form)
  await act(async () =>
    form.dispatchEvent(
      new domWindow.Event('submit', {
        bubbles: true,
        cancelable: true,
      }) as unknown as Event
    )
  )
}

afterEach(async () => {
  apiClient.get = originalGet
  if (renderedDialog) {
    await act(async () => renderedDialog?.root.unmount())
    renderedDialog.queryClient.clear()
    renderedDialog.host.remove()
    renderedDialog = null
  }
  document.body.replaceChildren()
})

after(() => {
  domWindow.close()
})

describe('invitation relationship lookup dialog', () => {
  test('searches a promotion link, marks deleted invitees, and refetches each page', async () => {
    const requests: Array<Record<string, unknown>> = []
    const secondPage = deferred<{ data: unknown }>()
    apiClient.get = async (url, config) => {
      assert.equal(url, '/api/user/aff/search')
      const params = config?.params ?? {}
      requests.push(params)
      const page = Number(params.p)
      if (page === 2) return secondPage.promise
      return {
        data: {
          success: true,
          data: {
            aff_code: '46fD',
            owner: { ...user(1, 'affiliate-owner'), inviter_id: 7 },
            inviter: { ...user(7, 'upstream-inviter'), aff_code: 'UP01' },
            invitees: {
              items: [user(22, 'deleted-invitee', '2026-09-12T00:00:00Z')],
              total: 21,
              page,
              page_size: 20,
            },
          },
        },
      }
    }

    await renderDialog()
    assert.ok(
      document.body.textContent?.includes(
        'Search for a user to view their invitation code, inviter, and direct invitees.'
      )
    )
    assert.equal(
      document.querySelector('label[for="affiliate-lookup-query"]'),
      null
    )
    assert.equal(
      getLookupInput().placeholder,
      'Enter a user ID, email, invitation code, or promotion link'
    )
    assert.equal(
      getLookupInput().getAttribute('aria-label'),
      'Search by user ID, email, invitation code, or promotion link.'
    )

    const link = 'https://www.6789api.top/sign-up?aff=46fD'
    await changeInput(link)
    await submitLookup()
    await act(async () =>
      waitForCondition(
        () => document.body.textContent?.includes('affiliate-owner') === true,
        'owner result was not rendered'
      )
    )

    assert.equal(requests[0]?.q, link)
    assert.equal(requests[0]?.p, 1)
    assert.ok(document.body.textContent?.includes('upstream-inviter'))
    assert.ok(document.body.textContent?.includes('UP01'))
    assert.ok(document.body.textContent?.includes('deleted-invitee'))
    assert.ok(document.body.textContent?.includes('Deleted'))
    assert.ok(document.body.textContent?.includes('Page 1 of 2'))

    const nextButton = document.querySelector<HTMLButtonElement>(
      'button[aria-label="Next"]'
    )
    assert.ok(nextButton)
    await act(async () => nextButton.click())
    await act(async () =>
      waitForCondition(
        () => requests.length === 2,
        'second page was not requested'
      )
    )
    assert.equal(requests[1]?.p, 2)
    await act(async () =>
      secondPage.resolve({
        data: {
          success: true,
          data: {
            aff_code: '46fD',
            owner: { ...user(1, 'affiliate-owner'), inviter_id: 7 },
            inviter: { ...user(7, 'upstream-inviter'), aff_code: 'UP01' },
            invitees: {
              items: [user(2, 'active-invitee')],
              total: 21,
              page: 2,
              page_size: 20,
            },
          },
        },
      })
    )
    await act(async () =>
      waitForCondition(
        () => document.body.textContent?.includes('active-invitee') === true,
        'second page was not rendered'
      )
    )
    assert.ok(document.body.textContent?.includes('active-invitee'))

    const closeButton = document.querySelector<HTMLButtonElement>(
      '[data-slot="dialog-close"]'
    )
    assert.ok(closeButton)
    await act(async () => closeButton.click())
    await act(async () =>
      waitForCondition(
        () => document.querySelector('[data-slot="dialog-content"]') === null,
        'dialog did not close'
      )
    )
    const reopenButton =
      document.querySelector<HTMLButtonElement>('#reopen-dialog')
    assert.ok(reopenButton)
    await act(async () => reopenButton.click())
    await act(async () =>
      waitForCondition(
        () => document.querySelector('#affiliate-lookup-query') !== null,
        'dialog did not reopen'
      )
    )
    assert.equal(getLookupInput().value, '')
  })

  test('shows an empty state and repeats an identical database lookup', async () => {
    let requestCount = 0
    apiClient.get = async () => {
      requestCount += 1
      return {
        data: {
          success: true,
          data: {
            aff_code: 'missing',
            owner: null,
            invitees: { items: [], total: 0, page: 1, page_size: 20 },
          },
        },
      }
    }

    await renderDialog()
    await changeInput('missing')
    await submitLookup()
    await act(async () =>
      waitForCondition(
        () =>
          document.body.textContent?.includes(
            'No invitation relationship found'
          ) === true,
        'empty result was not rendered'
      )
    )
    assert.equal(requestCount, 1)

    await submitLookup()
    await act(async () =>
      waitForCondition(
        () => requestCount === 2,
        'identical lookup was served without a new request'
      )
    )
  })

  test('shows the server error when an invitation lookup fails', async () => {
    apiClient.get = async () => ({
      data: { success: false, message: 'lookup rejected' },
    })

    await renderDialog()
    await changeInput('46fD')
    await submitLookup()
    await act(async () =>
      waitForCondition(
        () => document.body.textContent?.includes('lookup rejected') === true,
        'lookup error was not rendered'
      )
    )

    assert.ok(
      document.body.textContent?.includes(
        'Failed to search invitation relationships'
      )
    )
  })

  test('localizes ambiguous automatic matches', async () => {
    let submittedQuery = ''
    apiClient.get = async (_url, config) => {
      submittedQuery = String(config?.params?.q ?? '')
      return {
        data: {
          success: false,
          code: 'affiliate_lookup_ambiguous',
          message: 'server fallback',
        },
      }
    }

    await renderDialog()
    await changeInput('collision')
    await submitLookup()
    await act(async () =>
      waitForCondition(
        () =>
          document.body.textContent?.includes(
            'Multiple users matched. Use id:, email:, or aff: to specify the lookup type.'
          ) === true,
        'ambiguous lookup guidance was not rendered'
      )
    )

    assert.equal(submittedQuery, 'collision')
    assert.equal(document.body.textContent?.includes('server fallback'), false)
  })

  test('shows the recorded inviter ID when the inviter was hard deleted', async () => {
    apiClient.get = async () => ({
      data: {
        success: true,
        data: {
          aff_code: 'SELF',
          owner: { ...user(9, 'orphaned-owner'), inviter_id: 404 },
          inviter: null,
          invitees: { items: [], total: 0, page: 1, page_size: 20 },
        },
      },
    })

    await renderDialog()
    await changeInput('9')
    await submitLookup()
    await act(async () =>
      waitForCondition(
        () =>
          document.body.textContent?.includes(
            'The recorded inviter no longer exists (ID: 404).'
          ) === true,
        'missing inviter state was not rendered'
      )
    )
  })
})
