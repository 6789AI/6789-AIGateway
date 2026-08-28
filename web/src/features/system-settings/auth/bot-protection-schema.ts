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

const botProtectionSchema = z.object({
  TurnstileCheckEnabled: z.boolean(),
  BotProtectionProvider: z.enum([
    'turnstile',
    'recaptcha',
    'geetest_v4',
    'cap',
  ]),
  BotProtectionLoginEnabled: z.boolean(),
  BotProtectionRegisterEnabled: z.boolean(),
  BotProtectionEmailVerificationEnabled: z.boolean(),
  BotProtectionPasswordResetEnabled: z.boolean(),
  BotProtectionCheckinEnabled: z.boolean(),
  TurnstileSiteKey: z.string(),
  TurnstileSecretKey: z.string(),
  ReCaptchaSiteKey: z.string(),
  ReCaptchaSecretKey: z.string(),
  GeeTestCaptchaId: z.string(),
  GeeTestSecretKey: z.string(),
  CapServerURL: z.string(),
  CapSiteKey: z.string(),
  CapSecretKey: z.string(),
})

export type BotProtectionFormValues = z.infer<typeof botProtectionSchema>
export type BotProtectionProvider =
  BotProtectionFormValues['BotProtectionProvider']
export type SecretConfigured = Record<BotProtectionProvider, boolean>

export function createBotProtectionSchema(
  translate: (key: string) => string,
  secretConfigured: SecretConfigured
) {
  return botProtectionSchema.superRefine((data, context) => {
    if (!data.TurnstileCheckEnabled) return

    if (
      !data.BotProtectionLoginEnabled &&
      !data.BotProtectionRegisterEnabled &&
      !data.BotProtectionEmailVerificationEnabled &&
      !data.BotProtectionPasswordResetEnabled &&
      !data.BotProtectionCheckinEnabled
    ) {
      context.addIssue({
        code: 'custom',
        path: ['TurnstileCheckEnabled'],
        message: translate('Enable at least one protected feature'),
      })
    }

    if (data.BotProtectionProvider === 'turnstile') {
      if (!data.TurnstileSiteKey.trim()) {
        context.addIssue({
          code: 'custom',
          path: ['TurnstileSiteKey'],
          message: translate('Site Key is required'),
        })
      }
      if (!data.TurnstileSecretKey.trim() && !secretConfigured.turnstile) {
        context.addIssue({
          code: 'custom',
          path: ['TurnstileSecretKey'],
          message: translate('Secret Key is required'),
        })
      }
    }

    if (data.BotProtectionProvider === 'recaptcha') {
      if (!data.ReCaptchaSiteKey.trim()) {
        context.addIssue({
          code: 'custom',
          path: ['ReCaptchaSiteKey'],
          message: translate('Site Key is required'),
        })
      }
      if (!data.ReCaptchaSecretKey.trim() && !secretConfigured.recaptcha) {
        context.addIssue({
          code: 'custom',
          path: ['ReCaptchaSecretKey'],
          message: translate('Secret Key is required'),
        })
      }
    }

    if (data.BotProtectionProvider === 'geetest_v4') {
      if (!data.GeeTestCaptchaId.trim()) {
        context.addIssue({
          code: 'custom',
          path: ['GeeTestCaptchaId'],
          message: translate('Captcha ID is required'),
        })
      }
      if (!data.GeeTestSecretKey.trim() && !secretConfigured.geetest_v4) {
        context.addIssue({
          code: 'custom',
          path: ['GeeTestSecretKey'],
          message: translate('Captcha Key is required'),
        })
      }
    }

    if (data.BotProtectionProvider === 'cap') {
      const parsedCapURL = z.url().safeParse(data.CapServerURL.trim())
      let capURLIsHTTP = false
      if (parsedCapURL.success) {
        const capURL = new URL(parsedCapURL.data)
        capURLIsHTTP =
          (capURL.protocol === 'http:' || capURL.protocol === 'https:') &&
          capURL.username === '' &&
          capURL.password === '' &&
          capURL.search === '' &&
          capURL.hash === ''
      }
      if (!capURLIsHTTP) {
        context.addIssue({
          code: 'custom',
          path: ['CapServerURL'],
          message: translate('Enter a valid HTTP or HTTPS URL'),
        })
      }
      if (!data.CapSiteKey.trim()) {
        context.addIssue({
          code: 'custom',
          path: ['CapSiteKey'],
          message: translate('Site Key is required'),
        })
      }
      if (!data.CapSecretKey.trim() && !secretConfigured.cap) {
        context.addIssue({
          code: 'custom',
          path: ['CapSecretKey'],
          message: translate('Secret Key is required'),
        })
      }
    }
  })
}
