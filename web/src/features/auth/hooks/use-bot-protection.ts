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
import i18next from 'i18next'
import { useState } from 'react'
import { toast } from 'sonner'

import type { BotProtectionProvider } from '@/components/bot-protection'
import { useStatus } from '@/hooks/use-status'

export type BotProtectionScope =
  | 'login'
  | 'register'
  | 'email_verification'
  | 'password_reset'
  | 'checkin'

const providers = new Set<BotProtectionProvider>([
  'turnstile',
  'recaptcha',
  'geetest_v4',
  'cap',
])

export function useBotProtection(scope: BotProtectionScope) {
  const { status } = useStatus()
  const [proof, setProof] = useState('')
  const configuredProvider = status?.bot_protection_provider
  const provider = providers.has(configuredProvider as BotProtectionProvider)
    ? (configuredProvider as BotProtectionProvider)
    : 'turnstile'
  const masterEnabled =
    status?.bot_protection_enabled ?? status?.turnstile_check ?? false
  const siteKey =
    status?.bot_protection_site_key ?? status?.turnstile_site_key ?? ''
  const capAPIEndpoint = status?.bot_protection_cap_api_endpoint ?? ''
  const providerConfigured =
    provider === 'cap' ? Boolean(capAPIEndpoint) : Boolean(siteKey)

  const scopeEnabled = status?.[`bot_protection_${scope}_enabled`] ?? true
  const isEnabled = Boolean(masterEnabled && scopeEnabled && providerConfigured)

  const validate = (): boolean => {
    if (isEnabled && !proof) {
      toast.info(
        i18next.t('Please wait a moment, human check is initializing...')
      )
      return false
    }
    return true
  }

  return {
    isEnabled,
    provider,
    siteKey,
    capAPIEndpoint,
    proof,
    setProof,
    validate,
  }
}
