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
import * as z from 'zod'

export type EmailSenderProvider = 'smtp' | 'bt_mail'
export type SmtpSecurityMode = 'none' | 'ssl_tls' | 'starttls'

export function getSmtpSecurityMode(values: {
  SMTPSSLEnabled: boolean
  SMTPStartTLSEnabled: boolean
}): SmtpSecurityMode {
  if (values.SMTPSSLEnabled) return 'ssl_tls'
  if (values.SMTPStartTLSEnabled) return 'starttls'
  return 'none'
}

function isHTTPURL(value: string): boolean {
  try {
    const parsed = new URL(value)
    return (
      (parsed.protocol === 'http:' || parsed.protocol === 'https:') &&
      parsed.username === '' &&
      parsed.password === '' &&
      parsed.search === '' &&
      parsed.hash === ''
    )
  } catch {
    return false
  }
}

export const createEmailSchema = (
  t: (key: string) => string,
  btMailPasswordConfigured: boolean
) =>
  z
    .object({
      EmailSenderProvider: z.enum(['smtp', 'bt_mail']),
      SMTPServer: z.string(),
      SMTPPort: z.string().refine((value) => {
        const trimmed = value.trim()
        if (!trimmed) return true
        return /^\d+$/.test(trimmed)
      }, t('Port must be a positive integer')),
      SMTPAccount: z.string(),
      SMTPFromName: z.string(),
      SMTPFrom: z.string().refine((value) => {
        const trimmed = value.trim()
        if (!trimmed) return true
        return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed)
      }, t('Enter a valid email or leave blank')),
      SMTPToken: z.string(),
      SMTPSSLEnabled: z.boolean(),
      SMTPStartTLSEnabled: z.boolean(),
      SMTPInsecureSkipVerify: z.boolean(),
      SMTPForceAuthLogin: z.boolean(),
      BTMailAPIURL: z.string(),
      BTMailFromName: z.string(),
      BTMailFrom: z.string().refine((value) => {
        const trimmed = value.trim()
        if (!trimmed) return true
        return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed)
      }, t('Enter a valid email or leave blank')),
      BTMailPassword: z.string(),
    })
    .superRefine((values, context) => {
      if (values.EmailSenderProvider !== 'bt_mail') return

      const apiURL = values.BTMailAPIURL.trim()
      if (!apiURL) {
        context.addIssue({
          code: 'custom',
          path: ['BTMailAPIURL'],
          message: t('BT Mail API URL is required'),
        })
      } else if (!isHTTPURL(apiURL)) {
        context.addIssue({
          code: 'custom',
          path: ['BTMailAPIURL'],
          message: t('Enter a valid HTTP or HTTPS URL'),
        })
      }

      if (!values.BTMailFrom.trim()) {
        context.addIssue({
          code: 'custom',
          path: ['BTMailFrom'],
          message: t('Sender email is required'),
        })
      }
      if (!btMailPasswordConfigured && !values.BTMailPassword.trim()) {
        context.addIssue({
          code: 'custom',
          path: ['BTMailPassword'],
          message: t('Mailbox password is required'),
        })
      }
    })

export type EmailFormValues = z.infer<ReturnType<typeof createEmailSchema>>
export type EmailOptionUpdate = {
  key: keyof EmailFormValues
  value: string | boolean
}

function normalizeEmailFormValues(values: EmailFormValues): EmailFormValues {
  const securityMode = getSmtpSecurityMode(values)
  return {
    EmailSenderProvider: values.EmailSenderProvider,
    SMTPServer: values.SMTPServer.trim(),
    SMTPPort: values.SMTPPort.trim(),
    SMTPAccount: values.SMTPAccount.trim(),
    SMTPFromName: values.SMTPFromName.trim(),
    SMTPFrom: values.SMTPFrom.trim(),
    SMTPToken: values.SMTPToken.trim(),
    SMTPSSLEnabled: securityMode === 'ssl_tls',
    SMTPStartTLSEnabled: securityMode === 'starttls',
    SMTPInsecureSkipVerify: values.SMTPInsecureSkipVerify,
    SMTPForceAuthLogin: values.SMTPForceAuthLogin,
    BTMailAPIURL: values.BTMailAPIURL.trim(),
    BTMailFromName: values.BTMailFromName.trim(),
    BTMailFrom: values.BTMailFrom.trim(),
    BTMailPassword: values.BTMailPassword.trim(),
  }
}

export function buildEmailOptionUpdates(
  values: EmailFormValues,
  initialValues: EmailFormValues
): EmailOptionUpdate[] {
  const sanitized = normalizeEmailFormValues(values)
  const initial = normalizeEmailFormValues(initialValues)
  const configKeys: Array<Exclude<keyof EmailFormValues, 'EmailSenderProvider'>> = [
    'SMTPServer',
    'SMTPPort',
    'SMTPAccount',
    'SMTPFromName',
    'SMTPFrom',
    'SMTPToken',
    'SMTPSSLEnabled',
    'SMTPStartTLSEnabled',
    'SMTPInsecureSkipVerify',
    'SMTPForceAuthLogin',
    'BTMailAPIURL',
    'BTMailFromName',
    'BTMailFrom',
    'BTMailPassword',
  ]
  const updates: EmailOptionUpdate[] = []

  for (const key of configKeys) {
    const value = sanitized[key]
    if ((key === 'SMTPToken' || key === 'BTMailPassword') && value === '') {
      continue
    }
    if (value !== initial[key]) {
      updates.push({ key, value })
    }
  }

  if (sanitized.EmailSenderProvider !== initial.EmailSenderProvider) {
    updates.push({
      key: 'EmailSenderProvider',
      value: sanitized.EmailSenderProvider,
    })
  }
  return updates
}
