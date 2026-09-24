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

import type { VisibilityState } from '@tanstack/react-table'
import { Window } from 'happy-dom'

import type { User } from '../../types'

const domWindow = new Window({ url: 'http://localhost/users' })
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
const { flexRender, getCoreRowModel, useReactTable } =
  await import('@tanstack/react-table')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { TooltipProvider } = await import('@/components/ui/tooltip')
const { useUsersColumns } = await import('../users-columns')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        Deleted: 'Deleted',
        Disabled: 'Disabled',
        Email: 'Email',
        Enabled: 'Enabled',
      },
    },
  },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const baseUser: User = {
  id: 7,
  username: 'ada',
  display_name: 'Ada Lovelace',
  email: 'ada@example.com',
  quota: 1000,
  used_quota: 250,
  request_count: 12,
  group: 'default',
  status: 2,
  role: 1,
}

function ColumnsHarness(props: {
  columnIds: string[]
  visibility?: VisibilityState
}) {
  const columns = useUsersColumns()
  const table = useReactTable({
    data: [baseUser],
    columns,
    getCoreRowModel: getCoreRowModel(),
    state: { columnVisibility: props.visibility ?? {} },
  })
  const cells = table.getRowModel().rows[0]?.getAllCells() ?? []

  return props.columnIds.map((columnId) => {
    const cell = cells.find((candidate) => candidate.column.id === columnId)
    return (
      <div key={columnId} data-test-column={columnId}>
        {cell
          ? flexRender(cell.column.columnDef.cell, cell.getContext())
          : null}
      </div>
    )
  })
}

async function renderColumns(
  columnIds: string[],
  visibility?: VisibilityState
) {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <TooltipProvider>
          <ColumnsHarness columnIds={columnIds} visibility={visibility} />
        </TooltipProvider>
      </I18nextProvider>
    )
  })

  return { container, root }
}

describe('user table columns', () => {
  after(() => {
    domWindow.close()
  })

  test('renders the user email when the email column is selected', async () => {
    const rendered = await renderColumns(['email'])

    assert.equal(rendered.container.textContent, 'ada@example.com')

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })

  test('shows the hidden status beside the username and opens its full label on click', async () => {
    const rendered = await renderColumns(['id', 'username'], { status: false })
    const usernameCell = rendered.container.querySelector(
      '[data-test-column="username"]'
    )
    const idCell = rendered.container.querySelector('[data-test-column="id"]')
    const statusButton = usernameCell?.querySelector<HTMLButtonElement>(
      '[data-user-status="2"]'
    )

    assert.ok(statusButton)
    assert.equal(statusButton.getAttribute('aria-label'), 'Disabled')
    assert.ok(statusButton.querySelector('.bg-neutral'))
    assert.equal(idCell?.querySelector('[data-user-status]'), null)

    await act(async () => statusButton.click())

    const tooltip = document.body.querySelector('[data-slot="tooltip-content"]')
    assert.equal(tooltip?.textContent, 'Disabled')

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })

  test('moves the hidden status beside the ID when the username is hidden', async () => {
    const rendered = await renderColumns(['id'], {
      status: false,
      username: false,
    })
    const statusButton = rendered.container.querySelector<HTMLButtonElement>(
      '[data-test-column="id"] [data-user-status="2"]'
    )

    assert.ok(statusButton)
    assert.equal(statusButton.getAttribute('aria-label'), 'Disabled')

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })

  test('does not duplicate the status when the status column is visible', async () => {
    const rendered = await renderColumns(['id', 'username'], { status: true })

    assert.equal(rendered.container.querySelector('[data-user-status]'), null)

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })
})
