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

import { TASK_ACTIONS, TASK_STATUS } from '../../../constants'
import type { TaskLog } from '../../../types'

const domWindow = new Window({ url: 'http://localhost/usage-logs/task' })
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
const { flexRender, getCoreRowModel, useReactTable } =
  await import('@tanstack/react-table')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { useTaskLogsColumns } = await import('../task-logs-columns')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const baseTaskLog: TaskLog = {
  id: 1,
  user_id: 1,
  platform: '',
  task_id: 'task_public_id',
  action: TASK_ACTIONS.TEXT_GENERATE,
  channel_id: 90,
  submit_time: 1_700_000_000,
  status: TASK_STATUS.SUCCESS,
}

function ColumnHarness(props: { log: TaskLog; columnID: string }) {
  const columns = useTaskLogsColumns(false)
  const table = useReactTable({
    data: [props.log],
    columns,
    getCoreRowModel: getCoreRowModel(),
  })
  const cell = table
    .getRowModel()
    .rows[0]?.getAllCells()
    .find((candidate) => candidate.column.id === props.columnID)

  return cell ? flexRender(cell.column.columnDef.cell, cell.getContext()) : null
}

async function renderColumn(log: TaskLog, columnID: string) {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <ColumnHarness log={log} columnID={columnID} />
      </I18nextProvider>
    )
  })

  return { container, root }
}

describe('task log columns', () => {
  after(() => {
    domWindow.close()
  })

  test('shows the channel number without exposing the provider platform', async () => {
    const rendered = await renderColumn(
      { ...baseTaskLog, platform: 'grsai' },
      'task_id'
    )

    assert.match(rendered.container.textContent ?? '', /90 · Text to Video/)
    assert.doesNotMatch(rendered.container.textContent ?? '', /grsai/i)

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })

  test('keeps audio preview available when the user payload omits platform', async () => {
    const rendered = await renderColumn(
      {
        ...baseTaskLog,
        action: TASK_ACTIONS.MUSIC,
        data: JSON.stringify([{ audio_url: 'https://example.com/audio.mp3' }]),
      },
      'fail_reason'
    )

    assert.match(rendered.container.textContent ?? '', /Click to preview audio/)

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })
})
