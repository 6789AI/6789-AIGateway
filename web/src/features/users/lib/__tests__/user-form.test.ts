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
import { test } from 'node:test'

import {
  transformFormDataToPayload,
  transformUserToFormDefaults,
  USER_FORM_DEFAULT_VALUES,
  userFormSchema,
} from '../user-form'

test('editing a user preserves multiple authorized groups in the update payload', () => {
  const defaults = transformUserToFormDefaults({
    id: 7,
    username: 'ada',
    display_name: 'Ada',
    quota: 0,
    used_quota: 0,
    request_count: 0,
    group: 'default',
    authorized_groups: ['vip', 'image'],
    status: 1,
    role: 1,
  })
  assert.deepEqual(defaults.authorized_groups, ['vip', 'image'])
  assert.deepEqual(transformFormDataToPayload(defaults, 7).authorized_groups, [
    'vip',
    'image',
  ])
  assert.equal(transformFormDataToPayload(defaults, 7).group, 'default')
  assert.equal(
    transformFormDataToPayload(USER_FORM_DEFAULT_VALUES).authorized_groups,
    undefined
  )
})

test('a form with more than 32 authorized groups fails validation', () => {
  const authorizedGroups = Array.from(
    { length: 33 },
    (_, index) => `group-${index}`
  )
  const result = userFormSchema.safeParse({
    ...USER_FORM_DEFAULT_VALUES,
    authorized_groups: authorizedGroups,
  })

  assert.equal(result.success, false)
})

test('changing the primary group removes it from additional grants in the update payload', () => {
  const form = {
    ...USER_FORM_DEFAULT_VALUES,
    username: 'ada',
    group: 'vip',
    authorized_groups: ['vip', 'image'],
  }

  assert.deepEqual(transformFormDataToPayload(form, 7).authorized_groups, [
    'image',
  ])
})
