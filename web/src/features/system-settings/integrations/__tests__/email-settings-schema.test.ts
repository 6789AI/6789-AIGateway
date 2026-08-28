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

import {
  buildEmailOptionUpdates,
  createEmailSchema,
  type EmailFormValues,
} from '../email-settings-schema'

const translate = (key: string) => key
const smtpValues: EmailFormValues = {
  EmailSenderProvider: 'smtp',
  SMTPServer: 'smtp.example.com',
  SMTPPort: '587',
  SMTPAccount: 'smtp-user',
  SMTPFromName: 'SMTP Sender',
  SMTPFrom: 'smtp@example.com',
  SMTPToken: '',
  SMTPSSLEnabled: false,
  SMTPStartTLSEnabled: true,
  SMTPInsecureSkipVerify: false,
  SMTPForceAuthLogin: false,
  BTMailAPIURL: '',
  BTMailFromName: '',
  BTMailFrom: '',
  BTMailPassword: '',
}

describe('sending settings validation and persistence', () => {
  test('requires complete BT Mail configuration when it is selected', () => {
    const result = createEmailSchema(translate, false).safeParse({
      ...smtpValues,
      EmailSenderProvider: 'bt_mail',
    })

    assert.equal(result.success, false)
    if (result.success) return
    assert.deepEqual(
      result.error.issues.map((issue) => issue.path[0]).sort(),
      ['BTMailAPIURL', 'BTMailFrom', 'BTMailPassword']
    )
  })

  test('writes BT Mail configuration before provider without clearing SMTP', () => {
    const updates = buildEmailOptionUpdates(
      {
        ...smtpValues,
        EmailSenderProvider: 'bt_mail',
        BTMailAPIURL:
          'https://panel.example.com:8888/mail_sys/send_mail_http.json',
        BTMailFromName: 'BT Sender',
        BTMailFrom: 'bt@example.com',
        BTMailPassword: 'mail-password',
      },
      smtpValues
    )

    assert.deepEqual(
      updates.map((update) => update.key),
      [
        'BTMailAPIURL',
        'BTMailFromName',
        'BTMailFrom',
        'BTMailPassword',
        'EmailSenderProvider',
      ]
    )
    assert.equal(
      updates.some((update) => update.key.startsWith('SMTP')),
      false
    )
  })

})
