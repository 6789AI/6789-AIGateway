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
import { describe, test } from 'node:test'

import { TASK_ACTIONS, TASK_STATUS } from '../../constants.ts'
import { getTaskVideoPreviewUrl } from '../task-video-preview.ts'

describe('task video preview', () => {
  test('shows the proxy preview for a successful video stored in result_url', () => {
    assert.equal(
      getTaskVideoPreviewUrl({
        action: TASK_ACTIONS.TEXT_GENERATE,
        status: TASK_STATUS.SUCCESS,
        task_id: 'task_preview',
        result_url: 'https://provider.example/video.mp4',
      }),
      '/v1/videos/task_preview/content'
    )
  })

  test('keeps preview support for legacy video URLs stored in fail_reason', () => {
    assert.equal(
      getTaskVideoPreviewUrl({
        action: TASK_ACTIONS.GENERATE,
        status: TASK_STATUS.SUCCESS,
        task_id: 'task_legacy',
        fail_reason: 'https://provider.example/legacy.mp4',
      }),
      '/v1/videos/task_legacy/content'
    )
  })

  test('does not show a preview for failed or unfinished video tasks', () => {
    assert.equal(
      getTaskVideoPreviewUrl({
        action: TASK_ACTIONS.TEXT_GENERATE,
        status: TASK_STATUS.FAILURE,
        task_id: 'task_failed',
        result_url: 'https://provider.example/video.mp4',
      }),
      undefined
    )
  })
})
