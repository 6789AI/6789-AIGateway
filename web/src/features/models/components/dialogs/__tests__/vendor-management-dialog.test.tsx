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
import { after, afterEach, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window({ url: 'http://localhost/models/metadata' })
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
  'matchMedia',
  'customElements',
] as const
for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const React = await import('react')
const { act } = React
Object.defineProperty(globalThis, 'React', { configurable: true, value: React })
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { api } = await import('@/lib/api')
const { VendorManagementDialog } = await import('../vendor-management-dialog')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const testGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
testGlobals.IS_REACT_ACT_ENVIRONMENT = true

type ApiClient = {
  get: (
    url: string,
    config?: { params?: Record<string, unknown> }
  ) => Promise<{ data: unknown }>
  delete: (url: string) => Promise<{ data: unknown }>
}
const apiClient = api as unknown as ApiClient
const originalGet = apiClient.get
const originalDelete = apiClient.delete
let rendered: {
  host: HTMLDivElement
  root: ReturnType<typeof createRoot>
  queryClient: InstanceType<typeof QueryClient>
} | null = null

async function renderDialog() {
  const host = document.createElement('div')
  document.body.append(host)
  const root = createRoot(host)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  rendered = { host, root, queryClient }
  await act(async () =>
    root.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <VendorManagementDialog
            open
            onOpenChange={() => {}}
            onCreate={() => {}}
            onEdit={() => {}}
          />
        </I18nextProvider>
      </QueryClientProvider>
    )
  )
}

async function waitFor(condition: () => boolean) {
  if (condition()) return
  await new Promise<void>((resolve, reject) => {
    const observer = new MutationObserver(() => {
      if (!condition()) return
      clearTimeout(timeout)
      observer.disconnect()
      resolve()
    })
    const timeout = setTimeout(() => {
      observer.disconnect()
      reject(
        new Error(
          `Expected UI state was not reached: ${document.body.textContent}`
        )
      )
    }, 1500)
    observer.observe(document, {
      childList: true,
      subtree: true,
      characterData: true,
    })
  })
}

function vendorResponse(
  items: Array<{ id: number; name: string; icon: string }>
) {
  return {
    data: {
      success: true,
      data: { items, total: items.length, page: 1, page_size: 20 },
    },
  }
}

afterEach(async () => {
  apiClient.get = originalGet
  apiClient.delete = originalDelete
  if (rendered) {
    await act(async () => rendered?.root.unmount())
    rendered.queryClient.clear()
    rendered.host.remove()
    rendered = null
  }
  document.body.replaceChildren()
})

after(() => domWindow.close())

test('deletes a selected vendor only after confirmation and refreshes the list', async () => {
  let exists = true
  const deleted: string[] = []
  apiClient.get = async (url) => {
    assert.equal(url, '/api/vendors/search')
    return vendorResponse(exists ? [{ id: 7, name: 'Muse', icon: 'Muse' }] : [])
  }
  apiClient.delete = async (url) => {
    deleted.push(url)
    exists = false
    return { data: { success: true } }
  }

  await renderDialog()
  await act(async () =>
    waitFor(
      () =>
        document.querySelector('[aria-label="Delete Vendor: Muse"]') !== null
    )
  )
  const deleteButton = document.querySelector<HTMLButtonElement>(
    '[aria-label="Delete Vendor: Muse"]'
  )
  assert.ok(deleteButton)
  await act(async () => deleteButton.click())
  assert.deepEqual(deleted, [])
  assert.ok(
    document.body.textContent?.includes(
      'The icon library will remain available.'
    )
  )

  const confirmButton = [
    ...document.querySelectorAll<HTMLButtonElement>('button'),
  ].find((button) => button.textContent?.trim() === 'Delete')
  assert.ok(confirmButton)
  await act(async () => confirmButton.click())
  await act(async () =>
    waitFor(
      () =>
        document.querySelector('[aria-label="Delete Vendor: Muse"]') === null
    )
  )
  assert.deepEqual(deleted, ['/api/vendors/7'])
})

test('keeps the vendor visible when deletion is rejected', async () => {
  apiClient.get = async () =>
    vendorResponse([{ id: 7, name: 'Muse', icon: 'Muse' }])
  apiClient.delete = async () => ({
    data: { success: false, message: 'Reassign related models first' },
  })

  await renderDialog()
  await act(async () =>
    waitFor(
      () =>
        document.querySelector('[aria-label="Delete Vendor: Muse"]') !== null
    )
  )
  const deleteButton = document.querySelector<HTMLButtonElement>(
    '[aria-label="Delete Vendor: Muse"]'
  )
  assert.ok(deleteButton)
  await act(async () => deleteButton.click())
  const confirmButton = [
    ...document.querySelectorAll<HTMLButtonElement>('button'),
  ].find((button) => button.textContent?.trim() === 'Delete')
  assert.ok(confirmButton)
  await act(async () => confirmButton.click())
  assert.ok(document.querySelector('[aria-label="Delete Vendor: Muse"]'))
  assert.ok(
    document.body.textContent?.includes(
      'The icon library will remain available.'
    )
  )
})
