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

const domWindow = new Window({ url: 'http://localhost/' })
const domGlobals = [
  'window',
  'document',
  'navigator',
  'location',
  'history',
  'localStorage',
  'HTMLElement',
  'HTMLInputElement',
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
const { QueryClient, QueryClientProvider } = await import(
  '@tanstack/react-query'
)
const {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} = await import('@tanstack/react-router')
const i18next = (await import('i18next')).default
const { initReactI18next } = await import('react-i18next')

await i18next.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const { UserAuthForm } = await import(
  '../../sign-in/components/user-auth-form'
)
const { SignUpForm } = await import('../../sign-up/components/sign-up-form')

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const status = {
  password_login_enabled: true,
  oauth_register_enabled: false,
  email_verification: true,
}

async function renderAuthForm(component: React.ReactNode) {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  queryClient.setQueryData(['status'], status)
  const rootRoute = createRootRoute({ component: () => component })
  const router = createRouter({
    routeTree: rootRoute,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })

  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    )
  })

  return { container, root, queryClient }
}

async function unmountAuthForm(
  rendered: Awaited<ReturnType<typeof renderAuthForm>>
) {
  await act(async () => rendered.root.unmount())
  rendered.queryClient.clear()
  rendered.container.remove()
}

function autocompleteFor(container: HTMLElement, name: string) {
  const input = container.querySelector<HTMLInputElement>(
    `input[name="${name}"]`
  )
  assert.ok(input, `expected input[name="${name}"]`)
  return input.autocomplete
}

describe('authentication credential autocomplete', () => {
  after(() => {
    domWindow.close()
  })

  test('marks sign-in credentials for password managers', async () => {
    const rendered = await renderAuthForm(<UserAuthForm />)

    assert.equal(
      rendered.container.querySelector('form')?.autocomplete,
      'on'
    )
    assert.equal(autocompleteFor(rendered.container, 'username'), 'username')
    assert.equal(
      autocompleteFor(rendered.container, 'password'),
      'current-password'
    )

    await unmountAuthForm(rendered)
  })

  test('marks sign-up credentials and verification fields', async () => {
    const rendered = await renderAuthForm(<SignUpForm />)

    assert.equal(
      rendered.container.querySelector('form')?.autocomplete,
      'on'
    )
    assert.equal(autocompleteFor(rendered.container, 'username'), 'username')
    assert.equal(
      autocompleteFor(rendered.container, 'password'),
      'new-password'
    )
    assert.equal(
      autocompleteFor(rendered.container, 'confirmPassword'),
      'new-password'
    )
    assert.equal(autocompleteFor(rendered.container, 'email'), 'email')
    assert.equal(
      autocompleteFor(rendered.container, 'verification_code'),
      'one-time-code'
    )

    await unmountAuthForm(rendered)
  })
})
