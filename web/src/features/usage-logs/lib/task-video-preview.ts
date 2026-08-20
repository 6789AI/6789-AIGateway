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
import { TASK_ACTIONS, TASK_STATUS } from '../constants.ts'
import type { TaskLog } from '../types'

const videoTaskActions = new Set<string>([
  TASK_ACTIONS.GENERATE,
  TASK_ACTIONS.TEXT_GENERATE,
  TASK_ACTIONS.FIRST_TAIL_GENERATE,
  TASK_ACTIONS.REFERENCE_GENERATE,
  TASK_ACTIONS.REMIX_GENERATE,
])

type TaskVideoPreviewLog = Pick<
  TaskLog,
  'action' | 'fail_reason' | 'result_url' | 'status' | 'task_id'
>

export function getTaskVideoPreviewUrl(
  log: TaskVideoPreviewLog
): string | undefined {
  if (log.status !== TASK_STATUS.SUCCESS || !videoTaskActions.has(log.action)) {
    return undefined
  }

  const taskId = log.task_id.trim()
  const resultUrl = log.result_url?.trim()
  const legacyResultUrl = log.fail_reason?.trim()
  const hasResultUrl = resultUrl !== undefined && resultUrl !== ''
  const hasLegacyResultUrl = legacyResultUrl?.startsWith('http') ?? false
  if (taskId === '' || (!hasResultUrl && !hasLegacyResultUrl)) {
    return undefined
  }

  return `/v1/videos/${encodeURIComponent(taskId)}/content`
}
