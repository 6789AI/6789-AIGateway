/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
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
import { after, test } from 'node:test'

import { Window } from 'happy-dom'

import type { UserProfile } from '../../types'

const domWindow = new Window({ url: 'http://localhost/profile' })
for (const key of [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'MouseEvent',
  'MutationObserver',
  'getComputedStyle',
  'requestAnimationFrame',
  'cancelAnimationFrame',
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
const { ProfileHeader } = await import('../profile-header')

const i18n = createInstance()
await i18n
  .use(initReactI18next)
  .init({ lng: 'en', resources: { en: { translation: {} } } })
;(
  globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true

const host = document.createElement('div')
document.body.append(host)
const root = createRoot(host)
after(() => {
  act(() => root.unmount())
  host.remove()
})

const profile: UserProfile = {
  id: 4,
  username: 'ada',
  display_name: 'Ada',
  role: 1,
  group: 'default',
  authorized_groups: ['vip', 'image'],
  quota: 0,
  used_quota: 0,
  request_count: 0,
  status: 1,
  aff_count: 0,
  aff_quota: 0,
  aff_history_quota: 0,
  created_time: 0,
}

test('profile shows each granted group beside the primary group', async () => {
  await act(async () =>
    root.render(
      <I18nextProvider i18n={i18n}>
        <ProfileHeader profile={profile} loading={false} />
      </I18nextProvider>
    )
  )
  assert.ok(host.textContent?.includes('default'))
  assert.ok(host.textContent?.includes('Authorized groups'))
  assert.ok(host.textContent?.includes('vip'))
  assert.ok(host.textContent?.includes('image'))

  await act(async () =>
    root.render(
      <I18nextProvider i18n={i18n}>
        <ProfileHeader
          profile={{ ...profile, authorized_groups: [] }}
          loading={false}
        />
      </I18nextProvider>
    )
  )
  assert.equal(host.textContent?.includes('Authorized groups'), false)
})
