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

const domWindow = new Window({ url: 'http://localhost/users' })
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
  'PointerEvent',
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

const { act, useState } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { AuthorizedGroupPicker } = await import('../authorized-group-picker')

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

function PickerHarness(props: { groups: string[]; initialValue?: string[] }) {
  const [value, setValue] = useState<string[]>(props.initialValue ?? [])
  return (
    <I18nextProvider i18n={i18n}>
      <AuthorizedGroupPicker
        groups={props.groups}
        primaryGroup='default'
        value={value}
        onChange={setValue}
      />
    </I18nextProvider>
  )
}

test('granting two hidden groups and revoking one keeps the other selected', async () => {
  await act(async () =>
    root.render(<PickerHarness groups={['default', 'vip', 'image']} />)
  )
  const group = document.querySelector('[role="group"]')
  assert.equal(
    group?.getAttribute('aria-label'),
    'Additional authorized groups'
  )
  assert.equal(group?.textContent?.includes('default'), false)

  const checkbox = (
    name: string
  ): { control: HTMLElement; input: HTMLInputElement } => {
    const label = [...document.querySelectorAll('label')].find(
      (item) => item.textContent === name
    )
    assert.ok(label)
    const control = label.querySelector<HTMLElement>('[role="checkbox"]')
    assert.ok(control)
    const input = label.querySelector<HTMLInputElement>(
      'input[type="checkbox"]'
    )
    assert.ok(input)
    return { control, input }
  }
  await act(async () => checkbox('vip').input.click())
  assert.equal(checkbox('vip').control.getAttribute('aria-checked'), 'true')
  await act(async () => checkbox('image').input.click())
  assert.equal(checkbox('vip').control.getAttribute('aria-checked'), 'true')
  assert.equal(checkbox('image').control.getAttribute('aria-checked'), 'true')

  await act(async () => checkbox('vip').input.click())
  assert.equal(checkbox('vip').control.getAttribute('aria-checked'), 'false')
  assert.equal(checkbox('image').control.getAttribute('aria-checked'), 'true')
})

test('a deleted authorized group remains visible and can be revoked', async () => {
  await act(async () =>
    root.render(
      <PickerHarness
        key='deleted-group'
        groups={['default', 'vip']}
        initialValue={['removed-group']}
      />
    )
  )

  const label = [...document.querySelectorAll('label')].find((item) =>
    item.textContent?.includes('removed-group')
  )
  assert.ok(label)
  assert.match(label.textContent ?? '', /Deleted/)
  const checkbox = label.querySelector<HTMLInputElement>(
    'input[type="checkbox"]'
  )
  assert.ok(checkbox)
  assert.equal(checkbox.checked, true)

  await act(async () => checkbox.click())
  assert.equal(checkbox.checked, false)
})

test('a full grant list prevents selecting a 33rd group but allows revocation', async () => {
  const selected = Array.from({ length: 32 }, (_, index) => `group-${index}`)
  await act(async () =>
    root.render(
      <PickerHarness
        key='group-limit'
        groups={['default', ...selected, 'extra']}
        initialValue={selected}
      />
    )
  )

  const extra = [...document.querySelectorAll('label')].find(
    (label) => label.textContent === 'extra'
  )
  const selectedGroup = [...document.querySelectorAll('label')].find(
    (label) => label.textContent === 'group-0'
  )
  assert.ok(extra)
  assert.ok(selectedGroup)
  const extraCheckbox = extra.querySelector<HTMLInputElement>(
    'input[type="checkbox"]'
  )
  const selectedCheckbox = selectedGroup.querySelector<HTMLInputElement>(
    'input[type="checkbox"]'
  )
  assert.ok(extraCheckbox)
  assert.ok(selectedCheckbox)
  assert.equal(extraCheckbox.disabled, true)
  assert.equal(selectedCheckbox.disabled, false)

  await act(async () => selectedCheckbox.click())
  assert.equal(extraCheckbox.disabled, false)
})
